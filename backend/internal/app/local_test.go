package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kukusuyi/Questrace/backend/internal/config"
)

func request(t *testing.T, h http.Handler, method, path, token string, body any) (int, map[string]any) {
	t.Helper()
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(method, path, bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var response map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	return w.Code, response
}
func newTestRuntime(t *testing.T) *LocalRuntime {
	t.Helper()
	cfg, err := config.LoadLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	rt, err := NewLocal(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rt.DB.Close() })
	return rt
}
func setupAndLogin(t *testing.T, rt *LocalRuntime) string {
	t.Helper()
	code, res := request(t, rt, "POST", "/api/v1/system/setup", "", map[string]string{"token": rt.Config().SetupToken, "username": "owner", "password": "password123", "email": "owner@test.local", "education_stage": "university"})
	if code != 200 {
		t.Fatalf("setup: %d %v", code, res)
	}
	code, res = request(t, rt, "POST", "/api/v1/auth/login", "", map[string]string{"username": "owner", "password": "password123"})
	if code != 200 {
		t.Fatalf("login: %d %v", code, res)
	}
	return res["data"].(map[string]any)["token"].(string)
}
func TestLocalFirstSetupAndCRUD(t *testing.T) {
	rt := newTestRuntime(t)
	code, _ := request(t, rt, "POST", "/api/v1/system/setup", "", map[string]string{"token": "wrong"})
	if code != 403 {
		t.Fatalf("setup token: %d", code)
	}
	admin := setupAndLogin(t, rt)
	code, _ = request(t, rt, "POST", "/api/v1/system/setup", "", map[string]string{})
	if code != 409 {
		t.Fatal("setup reusable")
	}
	code, res := request(t, rt, "POST", "/api/v1/admin/users", admin, map[string]string{"username": "other", "password": "password123", "email": "other@test.local", "education_stage": "university"})
	if code != 200 {
		t.Fatalf("user: %v", res)
	}
	_, res = request(t, rt, "POST", "/api/v1/auth/login", "", map[string]string{"username": "other", "password": "password123"})
	other := res["data"].(map[string]any)["token"].(string)
	code, _ = request(t, rt, "GET", "/api/v1/admin/settings", other, nil)
	if code != 403 {
		t.Fatal("non-admin settings allowed")
	}
	q := map[string]any{"subject": "考研数学", "subject_id": "math_grad", "source_type": "manual", "question_json": map[string]string{"question_core": "1+1?"}, "semantic_summary": "", "mastery_status": "unmastered", "tags": map[string]any{}}
	code, res = request(t, rt, "POST", "/api/v1/wrong-questions", admin, q)
	if code != 200 {
		t.Fatalf("create without models: %d %v", code, res)
	}
	code, res = request(t, rt, "GET", "/api/v1/wrong-questions/1", admin, nil)
	if code != 200 {
		t.Fatalf("detail: %d %v", code, res)
	}
	code, _ = request(t, rt, "GET", "/api/v1/wrong-questions/1", other, nil)
	if code != 404 {
		t.Fatal("cross-user question")
	}
	code, res = request(t, rt, "GET", "/api/v1/vector-jobs", admin, nil)
	if code != 200 || res["data"].(map[string]any)["pending"] != float64(1) {
		t.Fatalf("durable queue: %v", res)
	}
	code, res = request(t, rt, "GET", "/api/v1/vector-jobs", other, nil)
	if code != 200 || len(res["data"].(map[string]any)) != 0 {
		t.Fatal("cross-user jobs")
	}
	code, _ = request(t, rt, "POST", "/api/v1/wrong-questions/1/similar", admin, map[string]any{"vector_type": "semantic"})
	if code != 503 {
		t.Fatalf("missing embedding status: %d", code)
	}
	code, res = request(t, rt, "GET", "/api/v1/dashboard/summary", admin, nil)
	if code != 200 {
		t.Fatalf("dashboard: %v", res)
	}
	q["source_image_id"] = 999
	q["source_image_url"] = "/api/v1/files/content/foreign"
	code, _ = request(t, rt, "POST", "/api/v1/wrong-questions", admin, q)
	if code != 404 {
		t.Fatal("invalid image accepted")
	}
	var n int
	rt.DB.QueryRow("SELECT count(*) FROM wrong_question").Scan(&n)
	if n != 1 {
		t.Fatal("invalid image created question")
	}
}
func TestAssetsAndOrigin(t *testing.T) {
	rt := newTestRuntime(t)
	for _, path := range []string{"/", "/questions/42", "/healthz"} {
		w := httptest.NewRecorder()
		rt.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}
	r := httptest.NewRequest("POST", "http://localhost/api/v1/system/setup", strings.NewReader(`{}`))
	r.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	rt.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross-origin setup")
	}
}
func TestBackupRestoreAndNewerSchema(t *testing.T) {
	rt := newTestRuntime(t)
	setupAndLogin(t, rt)
	dir := rt.Config().DataDir
	rt.DB.Close()
	archive := filepath.Join(t.TempDir(), "backup.zip")
	if err := Backup(dir, archive); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "files", "later.txt"), []byte("later"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Restore(dir, archive); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "files", "later.txt")); !os.IsNotExist(err) {
		t.Fatal("restore didn't replace files")
	}
	cfg, err := config.LoadLocal(dir)
	if err != nil {
		t.Fatal(err)
	}
	again, err := NewLocal(cfg, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	if again.NeedsSetup() {
		t.Fatal("account lost")
	}
	_, err = again.DB.ExecContext(context.Background(), "PRAGMA user_version=999")
	if err != nil {
		t.Fatal(err)
	}
	again.DB.Close()
	if newer, err := NewLocal(cfg, slog.Default()); err == nil {
		newer.DB.Close()
		t.Fatal("accepted newer schema")
	}
}

func TestFileOwnershipAndSettingsRedaction(t *testing.T) {
	rt := newTestRuntime(t)
	admin := setupAndLogin(t, rt)
	_, res := request(t, rt, "POST", "/api/v1/admin/users", admin, map[string]string{"username": "reader", "password": "password123", "email": "reader@test.local", "education_stage": "university"})
	if res["code"] != float64(0) {
		t.Fatal(res)
	}
	_, res = request(t, rt, "POST", "/api/v1/auth/login", "", map[string]string{"username": "reader", "password": "password123"})
	reader := res["data"].(map[string]any)["token"].(string)
	file := filepath.Join(rt.Config().File.Root, "test.png")
	if err := os.WriteFile(file, []byte("image bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := rt.DB.Exec("INSERT INTO file_record(user_id,storage_provider,bucket_name,object_key,file_name,file_url,file_size,mime_type,file_type) VALUES(1,'local','images','test.png','test.png','/api/v1/files/content/test.png',11,'image/png','image')")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		token string
		want  int
	}{{admin, 200}, {reader, 404}, {"", 401}} {
		code, _ := request(t, rt, "GET", "/api/v1/files/content/test.png", tc.token, nil)
		if code != tc.want {
			t.Fatalf("file auth: %d want %d", code, tc.want)
		}
	}
	code, _ := request(t, rt, "POST", "/api/v1/ocr/wrong-question-json", reader, map[string]any{"image_id": 1, "image_url": "http://private.invalid"})
	if code != 404 {
		t.Fatalf("OCR ownership: %d", code)
	}
	code, _ = request(t, rt, "POST", "/api/v1/ocr/wrong-question-json", reader, map[string]any{"image_id": 1, "image_url": "http://private.invalid", "purpose": "solution"})
	if code != 404 {
		t.Fatalf("solution OCR ownership: %d", code)
	}
	settings := map[string]any{"registration_enabled": false, "ocr": map[string]string{"name": "qwen", "model": "model", "api_key": "private-ocr-secret"}, "models": []any{}, "embedding": map[string]string{}, "download_url": ""}
	code, res = request(t, rt, "PUT", "/api/v1/admin/settings", admin, settings)
	if code != 200 {
		t.Fatalf("save settings: %v", res)
	}
	code, res = request(t, rt, "GET", "/api/v1/admin/settings", admin, nil)
	encoded, _ := json.Marshal(res)
	if code != 200 || bytes.Contains(encoded, []byte("private-ocr-secret")) || !bytes.Contains(encoded, []byte("__KEEP__")) {
		t.Fatalf("settings leaked secret: code=%d", code)
	}
	request(t, rt, "PUT", "/api/v1/admin/settings", admin, res["data"])
	if rt.Config().ImageOcr.APIKey != "private-ocr-secret" {
		t.Fatal("masked save discarded credential")
	}
}
func TestQuestionTagRollbackAndSoftDelete(t *testing.T) {
	rt := newTestRuntime(t)
	admin := setupAndLogin(t, rt)
	payload := map[string]any{"subject": "考研数学", "subject_id": "math_grad", "source_type": "manual", "question_json": map[string]string{"question_core": "q"}, "tags": map[string]any{"knowledge_points": []string{"calculus"}}, "mastery_status": "unmastered"}
	code, res := request(t, rt, "POST", "/api/v1/wrong-questions", admin, payload)
	if code != 200 {
		t.Fatalf("create: %v", res)
	}
	var usage int
	if err := rt.DB.QueryRow("SELECT usage_count FROM tag WHERE tag_name='calculus'").Scan(&usage); err != nil || usage != 1 {
		t.Fatalf("usage=%d %v", usage, err)
	}
	tx, err := rt.DB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec("DELETE FROM wrong_question_tag WHERE question_id=1"); err != nil {
		t.Fatal(err)
	}
	tx.Rollback()
	rt.DB.QueryRow("SELECT usage_count FROM tag WHERE tag_name='calculus'").Scan(&usage)
	if usage != 1 {
		t.Fatal("tag count escaped rollback")
	}
	code, res = request(t, rt, "DELETE", "/api/v1/wrong-questions/1", admin, nil)
	if code != 200 {
		t.Fatalf("delete: %v", res)
	}
	rt.DB.QueryRow("SELECT usage_count FROM tag WHERE tag_name='calculus'").Scan(&usage)
	if usage != 0 {
		t.Fatal("tag usage after delete", usage)
	}
}

func TestFileDatabaseFailureIsNotNotFound(t *testing.T) {
	rt := newTestRuntime(t)
	rt.DB.Close()
	w := httptest.NewRecorder()
	rt.file(w, httptest.NewRequest("GET", "/api/v1/files/content/test", nil), 1)
	if w.Code != 500 {
		t.Fatalf("database failure returned %d", w.Code)
	}
	w = httptest.NewRecorder()
	rt.jobs(w, httptest.NewRequest("GET", "/api/v1/vector-jobs/unknown", nil), 1)
	if w.Code != 404 {
		t.Fatalf("unknown task API returned %d", w.Code)
	}
}
