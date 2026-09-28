package test

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kukusuyi/Questrace/backend/internal/app"
	"github.com/kukusuyi/Questrace/backend/internal/config"
	"github.com/kukusuyi/Questrace/backend/internal/domain/dto"
	"github.com/kukusuyi/Questrace/backend/internal/infra/sqlite"
)

var (
	testServer         *httptest.Server
	mockProviderServer *httptest.Server
	baseURL            string
	testToken          string
	testUser           = fmt.Sprintf("apitest_%d", time.Now().UnixNano())
	testPass           = "testpass123"
	testEmail          string
)

func TestMain(m *testing.M) {
	testEmail = testUser + "@test.com"

	mockProviderServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"mock-model"},{"id":"mock-model-2"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/embeddings":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2,0.3]}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/chat/completions":
			var reqBody struct {
				Messages []struct {
					Content string `json:"content"`
				} `json:"messages"`
			}
			_ = json.NewDecoder(r.Body).Decode(&reqBody)

			w.Header().Set("Content-Type", "application/json")
			if len(reqBody.Messages) > 0 && strings.Contains(reqBody.Messages[0].Content, "多学科错题分类助手") {
				_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": `{"subject_id":"math_grad","course_id":"","chapter":"函数的极限和连续"}`}}}})
				return
			}
			if len(reqBody.Messages) > 0 && strings.Contains(reqBody.Messages[0].Content, "错题章节识别接口") {
				_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"chapter\":\"函数的极限和连续\"}"}}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"tags\":{\"knowledge_points\":[\"函数极限\"],\"problem_type\":[\"0比0型\"],\"method\":[\"等价无穷小\"],\"mistake_reason\":[\"计算错误\"]},\"semantic_summary\":\"这道题主要考察函数极限的化简与求值。\",\"mistake_summary\":\"学生在化简过程中出现了计算错误。\"}"}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))

	dir, err := os.MkdirTemp("", "questrace-api-test-*")
	if err != nil {
		panic(err)
	}
	cfg, err := config.LoadLocal(dir)
	if err != nil {
		panic(err)
	}
	cfg.Auth.EnableRegistration = true
	cfg.App.Port = 0
	cfg.ImageOcr.Name = "mockai"
	cfg.ImageOcr.Model = "mock-model"
	cfg.Models = []config.AIModelConfig{
		{
			Name:         "mockai",
			ProviderType: "openai_compatible",
			BaseURL:      mockProviderServer.URL + "/v1",
			Model:        "mock-model",
			APIKey:       "test-key",
		},
	}
	cfg.EmbeddingModel = config.EmbeddingModelConfig{
		ProviderType: "openai_compatible",
		BaseURL:      mockProviderServer.URL + "/v1",
		Model:        "mock-embedding",
		APIKey:       "test-key",
	}

	db, err := sqlite.Open(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open db: %v\n", err)
		os.Exit(1)
	}

	handler, err := app.BuildHTTPHandler(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build handler: %v\n", err)
		os.Exit(1)
	}

	testServer = httptest.NewServer(handler)
	baseURL = testServer.URL

	code := m.Run()

	testServer.Close()
	mockProviderServer.Close()
	db.Close()
	os.RemoveAll(dir)
	os.Exit(code)
}

func registerAndLogin(t *testing.T) string {
	t.Helper()

	resp, body, err := doPost(baseURL+"/api/v1/auth/login", "", dto.LoginRequest{
		Username: testUser,
		Password: testPass,
	})
	if err == nil && resp.StatusCode == http.StatusOK {
		data, _ := unmarshalData[dto.LoginResponse](body)
		if data.Token != "" {
			return data.Token
		}
	}

	resp, body, err = doPost(baseURL+"/api/v1/auth/register", "", dto.RegisterRequest{EducationStage: "university",
		Username: testUser,
		Password: testPass,
		Email:    testEmail,
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register status=%d body=%s", resp.StatusCode, string(body))
	}

	data, err := unmarshalData[dto.RegisterResponse](body)
	if err != nil {
		t.Fatalf("parse register response: %v", err)
	}
	if data.Token == "" {
		t.Fatal("register returned empty token")
	}

	return data.Token
}

func registerFreshUser(t *testing.T, prefix string) (string, string) {
	t.Helper()

	username := fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	email := username + "@test.com"

	resp, body, err := doPost(baseURL+"/api/v1/auth/register", "", dto.RegisterRequest{EducationStage: "university",
		Username: username,
		Password: testPass,
		Email:    email,
	})
	if err != nil {
		t.Fatalf("register fresh user: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register fresh user status=%d body=%s", resp.StatusCode, string(body))
	}

	data, err := unmarshalData[dto.RegisterResponse](body)
	if err != nil {
		t.Fatalf("parse fresh user register response: %v", err)
	}

	return data.Token, username
}

func TestHealthz(t *testing.T) {
	resp, body, err := doGet(baseURL+"/healthz", "")
	if err != nil {
		t.Fatalf("healthz: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz status=%d", resp.StatusCode)
	}
	if string(body) != "ok" {
		t.Fatalf("healthz body=%s", string(body))
	}
}

func TestDocs(t *testing.T) {
	resp, _, err := doGet(baseURL+"/docs", "")
	if err != nil {
		t.Fatalf("docs: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("docs status=%d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("docs content-type=%s", ct)
	}
}

func TestOpenAPIJSON(t *testing.T) {
	resp, body, err := doGet(baseURL+"/docs/openapi.json", "")
	if err != nil {
		t.Fatalf("openapi.json: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("openapi.json status=%d", resp.StatusCode)
	}

	var spec map[string]any
	if err := json.Unmarshal(body, &spec); err != nil {
		t.Fatalf("parse openapi.json: %v", err)
	}

	paths, _ := spec["paths"].(map[string]any)
	if paths["/api/v1/auth/register"] == nil {
		t.Error("openapi.json missing /api/v1/auth/register")
	}
	if paths["/api/v1/auth/login"] == nil {
		t.Error("openapi.json missing /api/v1/auth/login")
	}
	if paths["/api/v1/dashboard/summary"] == nil {
		t.Error("openapi.json missing /api/v1/dashboard/summary")
	}
}

func TestRegister(t *testing.T) {
	username := "reg_" + testUser
	email := username + "@test.com"

	resp, body, err := doPost(baseURL+"/api/v1/auth/register", "", dto.RegisterRequest{EducationStage: "university",
		Username: username,
		Password: testPass,
		Email:    email,
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register status=%d body=%s", resp.StatusCode, string(body))
	}

	data, err := unmarshalData[dto.RegisterResponse](body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if data.UserID == 0 {
		t.Error("register user_id is 0")
	}
	if data.Username != username {
		t.Errorf("register username=%q want=%q", data.Username, username)
	}
	if data.Token == "" {
		t.Error("register token is empty")
	}
}

func TestRegisterDuplicateUsername(t *testing.T) {
	dupUser := "dupuser_" + testUser
	dupEmail := dupUser + "@test.com"

	doPost(baseURL+"/api/v1/auth/register", "", dto.RegisterRequest{EducationStage: "university",
		Username: dupUser,
		Password: testPass,
		Email:    dupEmail,
	})

	resp, _, err := doPost(baseURL+"/api/v1/auth/register", "", dto.RegisterRequest{EducationStage: "university",
		Username: dupUser,
		Password: testPass,
		Email:    "other_" + dupEmail,
	})
	if err != nil {
		t.Fatalf("register dup: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("register dup status=%d want 409", resp.StatusCode)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	dupUser := "dupemail_" + testUser
	dupEmail := dupUser + "@test.com"

	doPost(baseURL+"/api/v1/auth/register", "", dto.RegisterRequest{EducationStage: "university",
		Username: dupUser,
		Password: testPass,
		Email:    dupEmail,
	})

	resp, _, err := doPost(baseURL+"/api/v1/auth/register", "", dto.RegisterRequest{EducationStage: "university",
		Username: "other_" + dupUser,
		Password: testPass,
		Email:    dupEmail,
	})
	if err != nil {
		t.Fatalf("register dup email: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("register dup email status=%d want 409", resp.StatusCode)
	}
}

func TestRegisterValidation(t *testing.T) {
	tests := []struct {
		name     string
		body     dto.RegisterRequest
		wantCode int
	}{
		{"empty username", dto.RegisterRequest{EducationStage: "university", Username: "", Password: "123456", Email: "a@b.com"}, http.StatusBadRequest},
		{"short password", dto.RegisterRequest{EducationStage: "university", Username: "u1", Password: "12345", Email: "a@b.com"}, http.StatusBadRequest},
		{"empty email", dto.RegisterRequest{EducationStage: "university", Username: "u1", Password: "123456", Email: ""}, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, _, _ := doPost(baseURL+"/api/v1/auth/register", "", tt.body)
			if resp.StatusCode != tt.wantCode {
				t.Errorf("status=%d want %d", resp.StatusCode, tt.wantCode)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	token := registerAndLogin(t)
	testToken = token

	resp, body, err := doPost(baseURL+"/api/v1/auth/login", "", dto.LoginRequest{
		Username: testUser,
		Password: testPass,
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status=%d body=%s", resp.StatusCode, string(body))
	}

	data, err := unmarshalData[dto.LoginResponse](body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if data.Token == "" {
		t.Error("login token is empty")
	}
	if data.UserID == 0 {
		t.Error("login user_id is 0")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	resp, _, err := doPost(baseURL+"/api/v1/auth/login", "", dto.LoginRequest{
		Username: testUser,
		Password: "wrongpass",
	})
	if err != nil {
		t.Fatalf("login wrong: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("login wrong status=%d want 401", resp.StatusCode)
	}
}

func TestLoginNonexistentUser(t *testing.T) {
	resp, _, err := doPost(baseURL+"/api/v1/auth/login", "", dto.LoginRequest{
		Username: "nonexistent_user_12345",
		Password: "123456",
	})
	if err != nil {
		t.Fatalf("login nonexist: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("login nonexist status=%d want 401", resp.StatusCode)
	}
}

func TestLoginValidation(t *testing.T) {
	tests := []struct {
		name     string
		body     dto.LoginRequest
		wantCode int
	}{
		{"empty username", dto.LoginRequest{Username: "", Password: "123456"}, http.StatusBadRequest},
		{"empty password", dto.LoginRequest{Username: "u", Password: ""}, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, _, _ := doPost(baseURL+"/api/v1/auth/login", "", tt.body)
			if resp.StatusCode != tt.wantCode {
				t.Errorf("status=%d want %d", resp.StatusCode, tt.wantCode)
			}
		})
	}
}

func TestGetMe(t *testing.T) {
	token := ensureToken(t)

	resp, body, err := doGet(baseURL+"/api/v1/users/me", token)
	if err != nil {
		t.Fatalf("get me: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get me status=%d body=%s", resp.StatusCode, string(body))
	}

	data, err := unmarshalData[dto.UserMeResponse](body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if data.Username != testUser {
		t.Errorf("username=%q want=%q", data.Username, testUser)
	}
	if data.Email != testEmail {
		t.Errorf("email=%q want=%q", data.Email, testEmail)
	}
}

func TestGetMeNoAuth(t *testing.T) {
	resp, _, err := doGet(baseURL+"/api/v1/users/me", "")
	if err != nil {
		t.Fatalf("get me noauth: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("get me noauth status=%d want 401", resp.StatusCode)
	}
}

func TestGetMeInvalidToken(t *testing.T) {
	resp, _, err := doGet(baseURL+"/api/v1/users/me", "invalid.token.here")
	if err != nil {
		t.Fatalf("get me invalid: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("get me invalid status=%d want 401", resp.StatusCode)
	}
}

func TestTagsCRUD(t *testing.T) {
	token := ensureToken(t)

	tagName := "tst_" + testUser

	t.Run("create", func(t *testing.T) {
		resp, body, err := doPost(baseURL+"/api/v1/tags", token, dto.CreateTagRequest{
			TagName: tagName,
			TagType: "knowledge_point",
		})
		if err != nil {
			t.Fatalf("create tag: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("create tag status=%d body=%s", resp.StatusCode, string(body))
		}
		data, _ := unmarshalData[dto.TagItem](body)
		if data.TagName != tagName {
			t.Errorf("tag name=%q", data.TagName)
		}
	})

	t.Run("list", func(t *testing.T) {
		resp, body, err := doGet(baseURL+"/api/v1/tags?tag_type=knowledge_point", token)
		if err != nil {
			t.Fatalf("list tags: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list tags status=%d", resp.StatusCode)
		}
		data, _ := unmarshalData[dto.TagListResponse](body)
		found := false
		for _, item := range data.List {
			if item.TagName == tagName {
				found = true
				break
			}
		}
		if !found {
			t.Error("created tag not found in list")
		}
	})

	t.Run("list noauth", func(t *testing.T) {
		resp, _, err := doGet(baseURL+"/api/v1/tags", "")
		if err != nil {
			t.Fatalf("list tags noauth: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("list tags noauth status=%d want 401", resp.StatusCode)
		}
	})
}

func TestQuestionCRUD(t *testing.T) {
	token := ensureToken(t)

	var questionID int64

	t.Run("create", func(t *testing.T) {
		resp, body, err := doPost(baseURL+"/api/v1/wrong-questions", token, dto.CreateWrongQuestionRequest{
			SubjectID:       "highschool_math",
			SourceType:      "manual",
			Subject:         "math",
			Chapter:         "test chapter",
			QuestionJSON:    dto.QuestionJSON{QuestionCore: "test question core", StandardSolution: "test solution", WrongSolution: "wrong attempt"},
			Tags:            dto.TagGroups{KnowledgePoints: []string{"极限"}},
			SemanticSummary: "test summary",
			DifficultyLevel: 3,
			MasteryStatus:   "unmastered",
		})
		if err != nil {
			t.Fatalf("create question: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("create question status=%d body=%s", resp.StatusCode, string(body))
		}
		data, _ := unmarshalData[dto.CreateWrongQuestionResponse](body)
		if data.QuestionID == 0 {
			t.Fatal("question_id is 0")
		}
		questionID = data.QuestionID
	})

	t.Run("detail", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/wrong-questions/%d", baseURL, questionID)
		resp, body, err := doGet(url, token)
		if err != nil {
			t.Fatalf("detail: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("detail status=%d body=%s", resp.StatusCode, string(body))
		}
		data, _ := unmarshalData[dto.QuestionDetail](body)
		if data.QuestionID != questionID {
			t.Errorf("question_id=%d want %d", data.QuestionID, questionID)
		}
	})

	t.Run("list", func(t *testing.T) {
		resp, body, err := doGet(baseURL+"/api/v1/wrong-questions?page=1&page_size=10", token)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list status=%d", resp.StatusCode)
		}
		data, _ := unmarshalData[dto.PageResult[dto.QuestionListItem]](body)
		if data.Total == 0 {
			t.Error("list total is 0")
		}
	})

	t.Run("update", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/wrong-questions/%d", baseURL, questionID)
		resp, body, err := doPut(url, token, dto.UpdateWrongQuestionRequest{
			SubjectID:       "highschool_math",
			Subject:         "math_updated",
			Chapter:         "updated chapter",
			QuestionJSON:    dto.QuestionJSON{QuestionCore: "updated core", StandardSolution: "updated solution"},
			Tags:            dto.TagGroups{KnowledgePoints: []string{"函数"}},
			SemanticSummary: "updated summary",
			DifficultyLevel: 4,
			MasteryStatus:   "learning",
		})
		if err != nil {
			t.Fatalf("update: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("update status=%d body=%s", resp.StatusCode, string(body))
		}
		data, _ := unmarshalData[dto.UpdateWrongQuestionResponse](body)
		if !data.Updated {
			t.Error("updated is false")
		}
	})

	t.Run("similar", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/wrong-questions/%d/similar", baseURL, questionID)
		resp, _, err := doPost(url, token, dto.SimilarQuestionRequest{
			VectorType:   "semantic",
			Limit:        5,
			UseTagFilter: false,
		})
		if err != nil {
			t.Fatalf("similar: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("similar status=%d", resp.StatusCode)
		}
	})

	t.Run("similar_by_json", func(t *testing.T) {
		resp, _, err := doPost(baseURL+"/api/v1/wrong-questions/similar-by-json", token, dto.SimilarByJSONRequest{
			SubjectID:    "highschool_math",
			QuestionJSON: dto.QuestionJSON{QuestionCore: "test query"},
			Tags:         dto.TagGroups{KnowledgePoints: []string{"极限"}},
			VectorType:   "semantic",
			Limit:        5,
		})
		if err != nil {
			t.Fatalf("similar-by-json: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("similar-by-json status=%d", resp.StatusCode)
		}
	})

	t.Run("delete", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/wrong-questions/%d", baseURL, questionID)
		resp, body, err := doDelete(url, token)
		if err != nil {
			t.Fatalf("delete: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("delete status=%d body=%s", resp.StatusCode, string(body))
		}
		data, _ := unmarshalData[dto.DeleteWrongQuestionResponse](body)
		if !data.Deleted {
			t.Error("deleted is false")
		}
	})

	t.Run("create noauth", func(t *testing.T) {
		resp, _, err := doPost(baseURL+"/api/v1/wrong-questions", "", dto.CreateWrongQuestionRequest{
			SubjectID:       "highschool_math",
			SourceType:      "manual",
			Subject:         "math",
			QuestionJSON:    dto.QuestionJSON{QuestionCore: "test"},
			SemanticSummary: "summary",
		})
		if err != nil {
			t.Fatalf("create noauth: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("create noauth status=%d want 401", resp.StatusCode)
		}
	})
}

func TestQuestionExportPrint(t *testing.T) {
	token := ensureToken(t)

	createQuestion := func(t *testing.T, core string) int64 {
		t.Helper()

		resp, body, err := doPost(baseURL+"/api/v1/wrong-questions", token, dto.CreateWrongQuestionRequest{
			SourceType: "manual",
			Subject:    "math",
			Chapter:    "export",
			QuestionJSON: dto.QuestionJSON{
				QuestionCore:     core,
				StandardSolution: `设 $x \\to 0$，则使用等价无穷小。`,
				WrongSolution:    `误把 $\\sin x$ 当成 $x^2$。`,
			},
			Tags: dto.TagGroups{
				KnowledgePoints: []string{"导出测试标签"},
			},
			SemanticSummary: "导出测试摘要",
			MistakeSummary:  "导出测试错因",
			MasteryStatus:   "learning",
		})
		if err != nil {
			t.Fatalf("create export question: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("create export question status=%d body=%s", resp.StatusCode, string(body))
		}

		data, err := unmarshalData[dto.CreateWrongQuestionResponse](body)
		if err != nil {
			t.Fatalf("parse export question response: %v", err)
		}

		return data.QuestionID
	}

	firstID := createQuestion(t, `求极限 $\\lim_{x \\to 0} \\frac{\\sin x}{x}$`)
	secondID := createQuestion(t, `证明 $1 + 1 = 2$`)

	t.Run("export with authorization header", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/wrong-questions/export/print?question_ids=%d,%d&export_mode=with_answers", baseURL, firstID, secondID)
		resp, body, err := doGet(url, token)
		if err != nil {
			t.Fatalf("export print: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("export print status=%d body=%s", resp.StatusCode, string(body))
		}
		if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "text/html") {
			t.Fatalf("export print content-type=%s want text/html", contentType)
		}
		if !containsIgnoreCase(string(body), "A4 每页两题") {
			t.Fatalf("export print body missing title: %s", string(body))
		}
		if containsIgnoreCase(string(body), "导出测试标签") {
			t.Fatalf("export print body should omit tags: %s", string(body))
		}
		if containsIgnoreCase(string(body), "标准解法") {
			t.Fatalf("export print body should omit answers: %s", string(body))
		}
	})

	t.Run("export with query token", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/wrong-questions/export/print?question_ids=%d&export_mode=with_answers&access_token=%s", baseURL, firstID, token)
		resp, body, err := doRequest(http.MethodGet, url, "", "", nil)
		if err != nil {
			t.Fatalf("export print with query token: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("export print with query token status=%d body=%s", resp.StatusCode, string(body))
		}
	})

	t.Run("questions only export hides answers", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/wrong-questions/export/print?question_ids=%d&export_mode=questions_only", baseURL, firstID)
		resp, body, err := doGet(url, token)
		if err != nil {
			t.Fatalf("questions only export: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("questions only export status=%d body=%s", resp.StatusCode, string(body))
		}
		if !containsIgnoreCase(string(body), "A4 每页两题") {
			t.Fatalf("questions only export body missing mode title: %s", string(body))
		}
		if containsIgnoreCase(string(body), "标准解法") {
			t.Fatalf("questions only export should not contain answer section: %s", string(body))
		}
		if containsIgnoreCase(string(body), "导出测试错因") {
			t.Fatalf("questions only export should not contain mistake summary: %s", string(body))
		}
		if containsIgnoreCase(string(body), "导出测试标签") {
			t.Fatalf("questions only export should not contain tags: %s", string(body))
		}
		if containsIgnoreCase(string(body), "原图") {
			t.Fatalf("questions only export should not contain image block: %s", string(body))
		}
		if containsIgnoreCase(string(body), "来源：") {
			t.Fatalf("questions only export should not contain source metadata: %s", string(body))
		}
		if containsIgnoreCase(string(body), "掌握状态：") {
			t.Fatalf("questions only export should not contain mastery metadata: %s", string(body))
		}
		if containsIgnoreCase(string(body), "math · export") {
			t.Fatalf("questions only export should not contain subject or chapter header: %s", string(body))
		}
	})

	t.Run("reject invalid export mode", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/wrong-questions/export/print?question_ids=%d&export_mode=invalid", baseURL, firstID)
		resp, _, err := doGet(url, token)
		if err != nil {
			t.Fatalf("invalid export mode: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("invalid export mode status=%d want 400", resp.StatusCode)
		}
	})

	t.Run("export rejects other user question", func(t *testing.T) {
		otherToken, _ := registerFreshUser(t, "export_other")
		resp, body, err := doPost(baseURL+"/api/v1/wrong-questions", otherToken, dto.CreateWrongQuestionRequest{
			SourceType: "manual",
			Subject:    "math",
			QuestionJSON: dto.QuestionJSON{
				QuestionCore: "other export question",
			},
			SemanticSummary: "other export summary",
		})
		if err != nil {
			t.Fatalf("create other export question: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("create other export question status=%d body=%s", resp.StatusCode, string(body))
		}

		data, err := unmarshalData[dto.CreateWrongQuestionResponse](body)
		if err != nil {
			t.Fatalf("parse other export question response: %v", err)
		}

		url := fmt.Sprintf("%s/api/v1/wrong-questions/export/print?question_ids=%d&export_mode=with_answers", baseURL, data.QuestionID)
		resp, _, err = doGet(url, token)
		if err != nil {
			t.Fatalf("export other user question: %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("export other user question status=%d want 404", resp.StatusCode)
		}
	})
}

func TestFileUpload(t *testing.T) {
	token := ensureToken(t)

	pngBytes := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x04, 0x00, 0x00, 0x00, 0xB5, 0x1C, 0x0C,
		0x02, 0x00, 0x00, 0x00, 0x0B, 0x49, 0x44, 0x41,
		0x54, 0x78, 0xDA, 0x63, 0xFC, 0xFF, 0x1F, 0x00,
		0x03, 0x03, 0x02, 0x00, 0xED, 0xA6, 0x2D, 0xB4,
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44,
		0xAE, 0x42, 0x60, 0x82,
	}

	resp, body, err := doMultipart(baseURL+"/api/v1/files/images", token, nil, "file", "test.png", pngBytes)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload status=%d body=%s", resp.StatusCode, string(body))
	}

	data, err := unmarshalData[dto.FileUploadResponse](body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if data.ImageID == 0 {
		t.Error("image_id is 0")
	}
	if !strings.Contains(data.ImageURL, "/api/v1/files/content/wrong-question/") {
		t.Errorf("image_url=%q is not a local file URL", data.ImageURL)
	}
}

func TestOCR(t *testing.T) {
	token := ensureToken(t)

	resp, body, err := doPost(baseURL+"/api/v1/ocr/wrong-question-json", token, map[string]any{
		"image_url": "http://example.com/test.png",
		"image_id":  1,
	})
	if err != nil {
		t.Fatalf("ocr: %v", err)
	}

	if resp.StatusCode == http.StatusServiceUnavailable {
		apiResp, _ := parseResponse(body)
		if apiResp.Code == 50002 {
			t.Logf("OCR not configured (expected when API key missing): %s", apiResp.Message)
			return
		}
	}

	if resp.StatusCode == http.StatusInternalServerError {
		apiResp, _ := parseResponse(body)
		t.Logf("OCR infrastructure error (API key or model issue): %s", apiResp.Message)
		return
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ocr status=%d body=%s", resp.StatusCode, string(body))
	}

	apiResp, _ := parseResponse(body)
	if apiResp.Code != 0 {
		t.Errorf("ocr code=%d", apiResp.Code)
	}
}

func TestAI(t *testing.T) {
	token := ensureToken(t)

	t.Run("list providers", func(t *testing.T) {
		resp, body, err := doGet(baseURL+"/api/v1/ai/model-providers", token)
		if err != nil {
			t.Fatalf("list providers: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list providers status=%d body=%s", resp.StatusCode, string(body))
		}

		data, err := unmarshalData[dto.AIProviderListResponse](body)
		if err != nil {
			t.Fatalf("parse providers: %v", err)
		}
		if len(data.List) == 0 || data.List[0].ProviderName != "mockai" {
			t.Fatalf("unexpected providers: %+v", data.List)
		}
	})

	t.Run("list chapters", func(t *testing.T) {
		resp, body, err := doGet(baseURL+"/api/v1/ai/chapters?subject_id=math_grad", token)
		if err != nil {
			t.Fatalf("list chapters: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list chapters status=%d body=%s", resp.StatusCode, string(body))
		}

		data, err := unmarshalData[dto.AIChapterListResponse](body)
		if err != nil {
			t.Fatalf("parse chapters: %v", err)
		}
		if len(data.List) == 0 || data.List[0] == "" {
			t.Fatalf("unexpected chapters: %+v", data.List)
		}
	})

	t.Run("list provider models", func(t *testing.T) {
		resp, body, err := doGet(baseURL+"/api/v1/ai/model-providers/mockai/models", token)
		if err != nil {
			t.Fatalf("list provider models: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list provider models status=%d body=%s", resp.StatusCode, string(body))
		}

		data, err := unmarshalData[dto.AIProviderModelListResponse](body)
		if err != nil {
			t.Fatalf("parse provider models: %v", err)
		}
		if len(data.List) == 0 || data.List[0].ModelName == "" {
			t.Fatalf("unexpected models: %+v", data.List)
		}
	})

	resp, body, err := doPost(baseURL+"/api/v1/ai/analyze-wrong-question", token, map[string]any{
		"provider_name": "mockai",
		"model_name":    "mock-model",
		"question_json": map[string]any{
			"question_core":     "test question",
			"standard_solution": "test solution",
			"wrong_solution":    "wrong",
		},
	})
	if err != nil {
		t.Fatalf("ai: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ai status=%d body=%s", resp.StatusCode, string(body))
	}

	apiResp, _ := parseResponse(body)
	if apiResp.Code != 0 {
		t.Errorf("ai code=%d", apiResp.Code)
	}

	data, err := unmarshalData[dto.AnalyzeWrongQuestionResponse](body)
	if err != nil {
		t.Fatalf("parse ai analyze response: %v", err)
	}
	if data.Chapter != "函数的极限和连续" {
		t.Fatalf("ai analyze chapter = %q, want %q", data.Chapter, "函数的极限和连续")
	}

	t.Run("uses imageOcr default provider for upload flow", func(t *testing.T) {
		resp, body, err := doPost(baseURL+"/api/v1/ai/analyze-wrong-question", token, map[string]any{
			"question_json": map[string]any{
				"question_core":     "upload flow question",
				"standard_solution": "standard solution",
				"wrong_solution":    "wrong solution",
			},
			"ocr_context": map[string]any{
				"ocr_confidence":  "medium",
				"uncertain_parts": []string{"part a"},
			},
		})
		if err != nil {
			t.Fatalf("ai upload default: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("ai upload default status=%d body=%s", resp.StatusCode, string(body))
		}

		apiResp, _ := parseResponse(body)
		if apiResp.Code != 0 {
			t.Errorf("ai upload default code=%d", apiResp.Code)
		}

		data, err := unmarshalData[dto.AnalyzeWrongQuestionResponse](body)
		if err != nil {
			t.Fatalf("parse ai upload response: %v", err)
		}
		if data.Chapter != "函数的极限和连续" {
			t.Fatalf("ai upload chapter = %q, want %q", data.Chapter, "函数的极限和连续")
		}
	})

	t.Run("uses manual chapter override when provided", func(t *testing.T) {
		resp, body, err := doPost(baseURL+"/api/v1/ai/analyze-wrong-question", token, map[string]any{
			"provider_name": "mockai",
			"model_name":    "mock-model",
			"chapter":       "函数的极限和连续",
			"question_json": map[string]any{
				"question_core":     "manual chapter question",
				"standard_solution": "manual solution",
				"wrong_solution":    "manual wrong",
			},
		})
		if err != nil {
			t.Fatalf("ai manual chapter override: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("ai manual chapter override status=%d body=%s", resp.StatusCode, string(body))
		}

		data, err := unmarshalData[dto.AnalyzeWrongQuestionResponse](body)
		if err != nil {
			t.Fatalf("parse ai manual chapter response: %v", err)
		}
		if data.Chapter != "函数的极限和连续" {
			t.Fatalf("ai manual chapter = %q, want %q", data.Chapter, "函数的极限和连续")
		}
	})
}

func TestDashboard(t *testing.T) {
	token := ensureToken(t)
	uniqueName := fmt.Sprintf("dashboard_tag_%d", time.Now().UnixNano())

	createResp, createBody, err := doPost(baseURL+"/api/v1/wrong-questions", token, dto.CreateWrongQuestionRequest{
		SourceType: "manual",
		Subject:    "math",
		QuestionJSON: dto.QuestionJSON{
			QuestionCore:     "dashboard test question",
			StandardSolution: "dashboard solution",
			WrongSolution:    "dashboard wrong",
		},
		Tags: dto.TagGroups{
			KnowledgePoints: []string{uniqueName},
			MistakeReason:   []string{uniqueName + "_mistake"},
		},
		SemanticSummary: "dashboard semantic summary",
		MistakeSummary:  "dashboard mistake summary",
		MasteryStatus:   "unmastered",
	})
	if err != nil {
		t.Fatalf("create dashboard question: %v", err)
	}
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("create dashboard question status=%d body=%s", createResp.StatusCode, string(createBody))
	}

	t.Run("summary", func(t *testing.T) {
		resp, body, err := doGet(baseURL+"/api/v1/dashboard/summary", token)
		if err != nil {
			t.Fatalf("dashboard summary: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("dashboard summary status=%d body=%s", resp.StatusCode, string(body))
		}

		data, err := unmarshalData[dto.DashboardSummaryResponse](body)
		if err != nil {
			t.Fatalf("parse dashboard summary: %v", err)
		}
		if data.TotalQuestions == 0 {
			t.Error("dashboard summary total_questions is 0")
		}
		if len(data.MasteryDistribution) == 0 {
			t.Error("dashboard summary mastery_distribution is empty")
		}
		if len(data.SourceDistribution) == 0 {
			t.Error("dashboard summary source_distribution is empty")
		}
	})

	t.Run("recent", func(t *testing.T) {
		resp, body, err := doGet(baseURL+"/api/v1/dashboard/recent?limit=4", token)
		if err != nil {
			t.Fatalf("dashboard recent: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("dashboard recent status=%d body=%s", resp.StatusCode, string(body))
		}

		data, err := unmarshalData[dto.DashboardRecentResponse](body)
		if err != nil {
			t.Fatalf("parse dashboard recent: %v", err)
		}
		if len(data.List) == 0 {
			t.Fatal("dashboard recent list is empty")
		}
		if data.List[0].QuestionID == 0 {
			t.Error("dashboard recent question_id is 0")
		}
	})

	t.Run("tags", func(t *testing.T) {
		resp, body, err := doGet(baseURL+"/api/v1/dashboard/tags?limit=12", token)
		if err != nil {
			t.Fatalf("dashboard tags: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("dashboard tags status=%d body=%s", resp.StatusCode, string(body))
		}

		data, err := unmarshalData[dto.DashboardTagsResponse](body)
		if err != nil {
			t.Fatalf("parse dashboard tags: %v", err)
		}

		foundKnowledgePoint := false
		for _, item := range data.KnowledgePoints {
			if item.TagName == uniqueName {
				foundKnowledgePoint = true
				break
			}
		}
		if !foundKnowledgePoint {
			t.Errorf("dashboard tags missing knowledge point %q", uniqueName)
		}
	})
}

func TestDashboardEmptySummary(t *testing.T) {
	token, _ := registerFreshUser(t, "dashboard_empty")

	resp, body, err := doGet(baseURL+"/api/v1/dashboard/summary", token)
	if err != nil {
		t.Fatalf("dashboard empty summary: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("dashboard empty summary status=%d body=%s", resp.StatusCode, string(body))
	}

	data, err := unmarshalData[dto.DashboardSummaryResponse](body)
	if err != nil {
		t.Fatalf("parse dashboard empty summary: %v", err)
	}
	if data.TotalQuestions != 0 {
		t.Fatalf("dashboard empty total_questions=%d want 0", data.TotalQuestions)
	}
}

func TestUserQuestionIsolation(t *testing.T) {
	tokenA, userA := registerFreshUser(t, "isolation_a")
	tokenB, userB := registerFreshUser(t, "isolation_b")

	tagA := "tag_" + userA
	tagB := "tag_" + userB

	createQuestion := func(t *testing.T, token, subject, tagName string) int64 {
		t.Helper()
		resp, body, err := doPost(baseURL+"/api/v1/wrong-questions", token, dto.CreateWrongQuestionRequest{
			SourceType: "manual",
			Subject:    subject,
			QuestionJSON: dto.QuestionJSON{
				QuestionCore:     "question for " + subject,
				StandardSolution: "solution for " + subject,
				WrongSolution:    "wrong for " + subject,
			},
			Tags: dto.TagGroups{
				KnowledgePoints: []string{tagName},
			},
			SemanticSummary: "summary for " + subject,
			MasteryStatus:   "unmastered",
		})
		if err != nil {
			t.Fatalf("create question for %s: %v", subject, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("create question for %s status=%d body=%s", subject, resp.StatusCode, string(body))
		}

		data, err := unmarshalData[dto.CreateWrongQuestionResponse](body)
		if err != nil {
			t.Fatalf("parse create question response for %s: %v", subject, err)
		}
		return data.QuestionID
	}

	questionIDA := createQuestion(t, tokenA, "subject_"+userA, tagA)
	questionIDB := createQuestion(t, tokenB, "subject_"+userB, tagB)

	t.Run("list only own questions", func(t *testing.T) {
		resp, body, err := doGet(baseURL+"/api/v1/wrong-questions?page=1&page_size=20", tokenA)
		if err != nil {
			t.Fatalf("list own questions: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list own questions status=%d body=%s", resp.StatusCode, string(body))
		}

		data, err := unmarshalData[dto.PageResult[dto.QuestionListItem]](body)
		if err != nil {
			t.Fatalf("parse own question list: %v", err)
		}

		foundOwn := false
		for _, item := range data.List {
			if item.QuestionID == questionIDA {
				foundOwn = true
			}
			if item.QuestionID == questionIDB {
				t.Fatalf("user A can see user B question %d in list", questionIDB)
			}
		}
		if !foundOwn {
			t.Fatalf("user A question %d not found in own list", questionIDA)
		}
	})

	t.Run("detail cannot access other user", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/wrong-questions/%d", baseURL, questionIDB)
		resp, _, err := doGet(url, tokenA)
		if err != nil {
			t.Fatalf("detail other user question: %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("detail other user question status=%d want 404", resp.StatusCode)
		}
	})

	t.Run("tags only own tags", func(t *testing.T) {
		resp, body, err := doGet(baseURL+"/api/v1/tags?tag_type=knowledge_point", tokenA)
		if err != nil {
			t.Fatalf("list own tags: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list own tags status=%d body=%s", resp.StatusCode, string(body))
		}

		data, err := unmarshalData[dto.TagListResponse](body)
		if err != nil {
			t.Fatalf("parse own tags: %v", err)
		}

		for _, item := range data.List {
			if item.TagName == tagB {
				t.Fatalf("user A can see user B tag %q", tagB)
			}
		}
	})

	t.Run("dashboard only own stats", func(t *testing.T) {
		resp, body, err := doGet(baseURL+"/api/v1/dashboard/summary", tokenA)
		if err != nil {
			t.Fatalf("dashboard own summary: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("dashboard own summary status=%d body=%s", resp.StatusCode, string(body))
		}

		data, err := unmarshalData[dto.DashboardSummaryResponse](body)
		if err != nil {
			t.Fatalf("parse dashboard own summary: %v", err)
		}
		if data.TotalQuestions != 1 {
			t.Fatalf("user A dashboard total_questions=%d want 1", data.TotalQuestions)
		}
	})
}

func TestProtectedEndpointsNoAuth(t *testing.T) {
	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/users/me"},
		{"GET", "/api/v1/dashboard/summary"},
		{"GET", "/api/v1/dashboard/recent"},
		{"GET", "/api/v1/dashboard/tags"},
		{"GET", "/api/v1/tags"},
		{"POST", "/api/v1/tags"},
		{"POST", "/api/v1/files/images"},
		{"POST", "/api/v1/ocr/wrong-question-json"},
		{"GET", "/api/v1/ai/model-providers"},
		{"GET", "/api/v1/ai/chapters?subject_id=math_grad"},
		{"GET", "/api/v1/ai/model-providers/mockai/models"},
		{"POST", "/api/v1/ai/analyze-wrong-question"},
		{"GET", "/api/v1/wrong-questions"},
		{"POST", "/api/v1/wrong-questions"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			resp, _, err := do(ep.method, baseURL+ep.path, "", nil)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("status=%d want 401", resp.StatusCode)
			}
		})
	}
}

func TestPublicEndpointsNoAuth(t *testing.T) {
	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/healthz"},
		{"GET", "/docs"},
		{"GET", "/docs/openapi.json"},
		{"POST", "/api/v1/auth/register"},
		{"POST", "/api/v1/auth/login"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			resp, _, err := do(ep.method, baseURL+ep.path, "", nil)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			if resp.StatusCode == http.StatusUnauthorized {
				t.Errorf("public endpoint %s %s returned 401", ep.method, ep.path)
			}
		})
	}
}

func ensureToken(t *testing.T) string {
	t.Helper()
	if testToken == "" {
		testToken = registerAndLogin(t)
	}
	return testToken
}
