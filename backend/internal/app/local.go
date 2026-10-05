package app

import (
	"bytes"
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kukusuyi/Questrace/backend/internal/discovery"
	v1 "github.com/kukusuyi/Questrace/backend/internal/http/handler/v1"
	"github.com/kukusuyi/Questrace/backend/internal/pkg/buildinfo"
	"golang.org/x/crypto/bcrypt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	assets "github.com/kukusuyi/Questrace/backend"
	"github.com/kukusuyi/Questrace/backend/internal/config"
	"github.com/kukusuyi/Questrace/backend/internal/domain/dto"
	"github.com/kukusuyi/Questrace/backend/internal/domain/model"
	ai "github.com/kukusuyi/Questrace/backend/internal/infra/ai"
	"github.com/kukusuyi/Questrace/backend/internal/infra/sqlite"
	apperrors "github.com/kukusuyi/Questrace/backend/internal/pkg/errors"
	"github.com/kukusuyi/Questrace/backend/internal/pkg/session"
	"github.com/kukusuyi/Questrace/backend/internal/repository"
	"github.com/kukusuyi/Questrace/backend/internal/service"
)

type LocalRuntime struct {
	mu          sync.RWMutex
	setupMu     sync.Mutex
	cfg         config.Config
	DB          *sql.DB
	api         http.Handler
	logger      *slog.Logger
	URLs        []string
	AddressList func() []string
	Discovery   func() discovery.Status
}

func NewLocal(cfg config.Config, logger *slog.Logger) (*LocalRuntime, error) {
	db, err := sqlite.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	// A settings write may outlive a process interrupted before jobs are requeued.
	// Reconcile persisted vectors on every startup so that restart repairs that gap.
	key := cfg.EmbeddingModel.ProviderType + "|" + strings.TrimRight(cfg.EmbeddingModel.BaseURL, "/") + "|" + cfg.EmbeddingModel.Model
	if _, err = db.Exec("UPDATE vector_job SET revision=revision+1,status='pending',next_attempt=0 WHERE question_id IN (SELECT question_id FROM local_vector WHERE model<>?)", key); err != nil {
		db.Close()
		return nil, err
	}
	api, err := BuildHTTPHandler(cfg, logger, db)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &LocalRuntime{cfg: cfg, DB: db, api: api, logger: logger}, nil
}
func (s *LocalRuntime) Config() config.Config { s.mu.RLock(); defer s.mu.RUnlock(); return s.cfg }
func (s *LocalRuntime) RunWorker(ctx context.Context) {
	(&service.LocalVector{DB: s.DB, Config: func() config.EmbeddingModelConfig { return s.Config().EmbeddingModel }}).Run(ctx)
}
func (s *LocalRuntime) NeedsSetup() bool {
	var n int
	return s.DB.QueryRow("SELECT count(*) FROM user WHERE role='admin'").Scan(&n) == nil && n == 0
}
func fail(w http.ResponseWriter, status int, message string) {
	dto.HandleError(w, apperrors.New(status, status*100, message))
}
func (s *LocalRuntime) user(r *http.Request) (model.User, error) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" {
		token = session.Read(r)
	}
	cfg := s.Config()
	repo := repository.NewSQLiteUserRepository(s.DB)
	auth := service.NewAuthService(repo, cfg.JWT, cfg.Auth)
	id, err := auth.ValidateJWT(token)
	if err != nil {
		return model.User{}, err
	}
	u, ok, err := repo.GetByID(id)
	if err != nil {
		return u, err
	}
	if !ok {
		return u, fmt.Errorf("unknown user")
	}
	return u, nil
}
func (s *LocalRuntime) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	// Browser mutations are same-origin. Native mobile clients send no Origin.
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host != r.Host {
			fail(w, 403, "不允许跨站请求")
			return
		}
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Cache-Control", "no-store")
		r.Body = http.MaxBytesReader(w, r.Body, 24*1024*1024)
	}
	switch r.URL.Path {
	case "/api/v1/updates/latest", "/api/v1/mobile/latest-version":
		s.updates(w, r)
		return
	case "/api/v1/auth/logout":
		if r.Method != "POST" {
			fail(w, 405, "Method not allowed")
			return
		}
		session.Clear(w)
		dto.WriteSuccess(w, true)
		return

	case "/api/v1/system/status":
		if r.Method != "GET" {
			fail(w, 405, "Method not allowed")
			return
		}
		cfg := s.Config()
		dto.WriteSuccess(w, map[string]any{"version": buildinfo.Version, "setup_required": s.NeedsSetup(), "registration_enabled": cfg.Auth.EnableRegistration, "ocr_enabled": cfg.ImageOcr.APIKey != "", "ai_enabled": len(cfg.Models) > 0, "embedding_enabled": cfg.EmbeddingModel.APIKey != "", "urls": s.addresses(), "discovery": s.discoveryStatus()})
		return
	case "/api/v1/system/setup":
		s.setup(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/v1/reviews/") || strings.HasPrefix(r.URL.Path, "/api/v1/admin/") || strings.HasPrefix(r.URL.Path, "/api/v1/vector-jobs") || strings.HasPrefix(r.URL.Path, "/api/v1/files/content/") {
		u, err := s.user(r)
		if err != nil {
			fail(w, 401, "请先登录")
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/reviews/") {
			(v1.ReviewHandler{Service: service.ReviewService{DB: s.DB}}).Serve(w, r, u.ID)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/admin/") {
			if u.Role != "admin" {
				fail(w, 403, "需要管理员权限")
				return
			}
			s.admin(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/files/content/") {
			s.file(w, r, u.ID)
			return
		}
		s.jobs(w, r, u.ID)
		return
	}
	if r.URL.Path == "/api/v1/ocr/wrong-question-json" && r.Method == "POST" {
		u, err := s.user(r)
		if err != nil {
			fail(w, 401, "请先登录")
			return
		}
		var req dto.OCRWrongQuestionRequest
		if dto.DecodeJSON(r, &req) != nil {
			fail(w, 400, "请求格式错误")
			return
		}
		rec, ok, err := repository.NewSQLiteFileRepository(s.DB).GetByID(req.ImageID)
		if err != nil {
			dto.HandleError(w, err)
			return
		}
		if !ok || rec.UserID != u.ID {
			fail(w, 404, "图片不存在")
			return
		}
		if !filepath.IsLocal(rec.ObjectKey) {
			fail(w, 400, "图片路径无效")
			return
		}
		data, err := os.ReadFile(filepath.Join(s.Config().File.Root, filepath.FromSlash(rec.ObjectKey)))
		if err != nil {
			dto.HandleError(w, err)
			return
		}
		req.ImageURL = "data:" + rec.MIMEType + ";base64," + base64.StdEncoding.EncodeToString(data)
		body, _ := json.Marshal(req)
		r.Body = io.NopCloser(bytes.NewReader(body))
	}
	if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/healthz" || strings.HasPrefix(r.URL.Path, "/docs") {
		s.mu.RLock()
		h := s.api
		s.mu.RUnlock()
		h.ServeHTTP(w, r)
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		fail(w, 405, "Method not allowed")
		return
	}
	web, _ := fs.Sub(assets.Files, "web")
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" {
		name = "index.html"
	}
	if _, err := fs.Stat(web, name); err != nil {
		if strings.HasPrefix(name, "assets/") {
			http.NotFound(w, r)
			return
		}
		data, _ := fs.ReadFile(web, "index.html")
		if len(data) == 0 {
			data = []byte("<!doctype html><title>Questrace</title><p>Build the web app with node scripts/build.mjs.</p>")
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
		return
	}
	http.FileServer(http.FS(web)).ServeHTTP(w, r)
}
func (s *LocalRuntime) setup(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		fail(w, 405, "Method not allowed")
		return
	}
	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	if !s.NeedsSetup() {
		fail(w, 409, "初始化已完成")
		return
	}
	var req struct {
		Token          string `json:"token"`
		EducationStage string `json:"education_stage"`
		Username       string `json:"username"`
		Password       string `json:"password"`
		Email          string `json:"email"`
	}
	if dto.DecodeJSON(r, &req) != nil {
		fail(w, 400, "请求格式错误")
		return
	}
	if subtle.ConstantTimeCompare([]byte(req.Token), []byte(s.Config().SetupToken)) != 1 {
		fail(w, 403, "初始化凭据无效，请查看电脑启动信息")
		return
	}
	if len(req.Password) < 8 || len(req.Password) > 72 || strings.TrimSpace(req.Username) == "" || req.Email == "" {
		fail(w, 400, "请填写用户名、邮箱及 8–72 字节密码")
		return
	}
	if err := service.ValidateEducationStage(req.EducationStage); err != nil {
		dto.HandleError(w, err)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		dto.HandleError(w, err)
		return
	}
	_, err = repository.NewSQLiteUserRepository(s.DB).Create(model.User{Username: strings.TrimSpace(req.Username), Email: req.Email, PasswordHash: string(hash), Role: "admin", EducationStage: req.EducationStage})
	if err != nil {
		dto.HandleError(w, err)
		return
	}
	dto.WriteSuccess(w, map[string]bool{"initialized": true})
}
func (s *LocalRuntime) file(w http.ResponseWriter, r *http.Request, uid int64) {
	if r.Method != "GET" && r.Method != "HEAD" {
		fail(w, 405, "Method not allowed")
		return
	}
	key := strings.TrimPrefix(r.URL.Path, "/api/v1/files/content/")
	if !filepath.IsLocal(key) {
		http.NotFound(w, r)
		return
	}
	var mime string
	if err := s.DB.QueryRow("SELECT mime_type FROM file_record WHERE object_key=? AND user_id=?", key, uid).Scan(&mime); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
		} else {
			dto.HandleError(w, err)
		}
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeFile(w, r, filepath.Join(s.Config().File.Root, filepath.FromSlash(key)))
}
func (s *LocalRuntime) jobs(w http.ResponseWriter, r *http.Request, uid int64) {
	if r.URL.Path != "/api/v1/vector-jobs" && r.URL.Path != "/api/v1/vector-jobs/retry" {
		fail(w, 404, "接口不存在")
		return
	}
	if r.URL.Path == "/api/v1/vector-jobs/retry" && r.Method != "POST" {
		fail(w, 405, "Method not allowed")
		return
	}
	if r.Method == "POST" {
		_, err := s.DB.Exec("UPDATE vector_job SET status='pending',next_attempt=0 WHERE question_id IN (SELECT id FROM wrong_question WHERE user_id=? AND is_deleted=0) AND status<>'done'", uid)
		if err != nil {
			dto.HandleError(w, err)
			return
		}
		dto.WriteSuccess(w, true)
		return
	}
	if r.Method != "GET" {
		fail(w, 405, "Method not allowed")
		return
	}
	rows, err := s.DB.Query("SELECT j.status,count(*) FROM vector_job j JOIN wrong_question q ON q.id=j.question_id WHERE q.user_id=? AND q.is_deleted=0 GROUP BY j.status", uid)
	if err != nil {
		dto.HandleError(w, err)
		return
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var status string
		var n int
		if err = rows.Scan(&status, &n); err != nil {
			dto.HandleError(w, err)
			return
		}
		counts[status] = n
	}
	dto.WriteSuccess(w, counts)
}

type settings struct {
	Registration bool                        `json:"registration_enabled"`
	OCR          config.ImageOcrConfig       `json:"ocr"`
	Models       []config.AIModelConfig      `json:"models"`
	Embedding    config.EmbeddingModelConfig `json:"embedding"`
	DownloadURL  string                      `json:"download_url"`
}

func (s *LocalRuntime) admin(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/v1/admin/settings/models" {
		s.probeModels(w, r)
		return
	}
	if r.URL.Path == "/api/v1/admin/users" {
		if r.Method == "POST" {
			var req dto.RegisterRequest
			if err := dto.DecodeJSON(r, &req); err != nil {
				dto.HandleError(w, err)
				return
			}
			cfg := s.Config()
			auth := service.NewAuthService(repository.NewSQLiteUserRepository(s.DB), cfg.JWT, config.AuthConfig{EnableRegistration: true})
			result, err := auth.Register(req)
			if err != nil {
				dto.HandleError(w, err)
				return
			}
			dto.WriteSuccess(w, map[string]any{"user_id": result.UserID, "username": result.Username})
			return
		}
		if r.Method == "GET" {
			rows, err := s.DB.Query("SELECT id,username,email,role FROM user ORDER BY id")
			if err != nil {
				dto.HandleError(w, err)
				return
			}
			defer rows.Close()
			items := []map[string]any{}
			for rows.Next() {
				var id int64
				var name, email, role string
				if err = rows.Scan(&id, &name, &email, &role); err != nil {
					dto.HandleError(w, err)
					return
				}
				items = append(items, map[string]any{"id": id, "username": name, "email": email, "role": role})
			}
			dto.WriteSuccess(w, items)
			return
		}
	}
	if r.URL.Path != "/api/v1/admin/settings" && r.URL.Path != "/api/v1/admin/settings/test" {
		http.NotFound(w, r)
		return
	}
	cfg := s.Config()
	if r.Method == "GET" {
		models := append([]config.AIModelConfig{}, cfg.Models...)
		for i := range models {
			if models[i].APIKey != "" {
				models[i].APIKey = "__KEEP__"
			}
		}
		ocr := cfg.ImageOcr
		if ocr.APIKey != "" {
			ocr.APIKey = "__KEEP__"
		}
		emb := cfg.EmbeddingModel
		if emb.APIKey != "" {
			emb.APIKey = "__KEEP__"
		}
		dto.WriteSuccess(w, settings{cfg.Auth.EnableRegistration, ocr, models, emb, cfg.MobileVersion.DownloadURL})
		return
	}
	if r.Method != "PUT" && r.Method != "POST" {
		fail(w, 405, "Method not allowed")
		return
	}
	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	cfg = s.Config()
	var req settings
	if dto.DecodeJSON(r, &req) != nil {
		fail(w, 400, "设置格式错误")
		return
	}
	if req.OCR.APIKey == "__KEEP__" {
		req.OCR.APIKey = cfg.ImageOcr.APIKey
	}
	if req.Embedding.APIKey == "__KEEP__" {
		req.Embedding.APIKey = cfg.EmbeddingModel.APIKey
	}
	for i := range req.Models {
		if req.Models[i].APIKey == "__KEEP__" {
			req.Models[i].APIKey = ""
			savedName := req.Models[i].SavedName
			if savedName == "" {
				savedName = req.Models[i].Name
			}
			for _, old := range cfg.Models {
				if old.Name == savedName {
					req.Models[i].APIKey = old.APIKey
				}
			}
		}
	}
	for i := range req.Models {
		req.Models[i].SavedName = ""
	}
	if req.DownloadURL != "" {
		u, err := url.Parse(req.DownloadURL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			fail(w, 400, "APK 地址必须为 HTTP(S) URL")
			return
		}
	}
	cfg.Auth.EnableRegistration = req.Registration
	cfg.ImageOcr = req.OCR
	cfg.Models = req.Models
	cfg.EmbeddingModel = req.Embedding
	cfg.MobileVersion.DownloadURL = req.DownloadURL
	if cfg.EmbeddingModel.APIKey != "" {
		if _, err := ai.NewEmbeddingClient(cfg.EmbeddingModel); err != nil {
			fail(w, 400, "Embedding 配置不完整")
			return
		}
	}
	h, err := BuildHTTPHandler(cfg, s.logger, s.DB)
	if err != nil {
		fail(w, 400, "模型配置不完整或重复，请检查模型名称、接口地址和密钥")
		return
	}
	if strings.HasSuffix(r.URL.Path, "/test") {
		s.testModels(w, r, cfg)
		return
	}
	old := s.Config()
	if err = config.SaveLocal(cfg); err != nil {
		dto.HandleError(w, err)
		return
	}
	s.mu.Lock()
	s.cfg = cfg
	s.api = h
	s.mu.Unlock()
	if old.EmbeddingModel.Model != cfg.EmbeddingModel.Model || old.EmbeddingModel.BaseURL != cfg.EmbeddingModel.BaseURL || old.EmbeddingModel.ProviderType != cfg.EmbeddingModel.ProviderType {
		_, err = s.DB.Exec("UPDATE vector_job SET revision=revision+1,status='pending',next_attempt=0")
		if err != nil {
			dto.HandleError(w, err)
			return
		}
	}
	dto.WriteSuccess(w, map[string]bool{"saved": true})
}
func (s *LocalRuntime) testModels(w http.ResponseWriter, r *http.Request, cfg config.Config) {
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	result := map[string]string{}
	if cfg.EmbeddingModel.APIKey != "" {
		c, _ := ai.NewEmbeddingClient(cfg.EmbeddingModel)
		_, err := c.Embed(ctx, "连接测试")
		if err != nil {
			result["embedding"] = apperrors.ProviderMessage(0, err.Error())
		} else {
			result["embedding"] = "连接成功"
		}
	}
	if cfg.ImageOcr.APIKey != "" {
		registry, err := ai.NewRegistry([]config.AIModelConfig{{Name: "ocr", ProviderType: "qwen", APIKey: cfg.ImageOcr.APIKey, Model: cfg.ImageOcr.Model}})
		if err != nil {
			result["ocr"] = "OCR 配置无效"
		} else {
			p, _ := registry.Provider("ocr")
			if _, err = p.ListModels(ctx); err != nil {
				result["ocr"] = apperrors.ProviderMessage(0, err.Error())
			} else {
				result["ocr"] = "服务连接成功"
			}
		}
	}
	registry, err := ai.NewRegistry(cfg.Models)
	if err != nil {
		fail(w, 400, "模型配置无效")
		return
	}
	for _, c := range registry.Providers() {
		p, _ := registry.Provider(c.Name)
		_, err := p.ListModels(ctx)
		if err != nil {
			result[c.Name] = apperrors.ProviderMessage(0, err.Error())
		} else {
			result[c.Name] = "连接成功"
		}
	}
	dto.WriteSuccess(w, result)
}

func (s *LocalRuntime) addresses() []string {
	if s.AddressList != nil {
		return s.AddressList()
	}
	return s.URLs
}

func (s *LocalRuntime) discoveryStatus() discovery.Status {
	if s.Discovery != nil {
		return s.Discovery()
	}
	return discovery.Status{State: "disabled"}
}
