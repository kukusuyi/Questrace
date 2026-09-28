package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kukusuyi/Questrace/backend/internal/config"
	"github.com/kukusuyi/Questrace/backend/internal/domain/model"
	"github.com/kukusuyi/Questrace/backend/internal/infra/sqlite"
	"github.com/kukusuyi/Questrace/backend/internal/repository"
)

func TestLocalVectorsOwnershipRetryAndRevision(t *testing.T) {
	db, err := sqlite.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec("INSERT INTO user(id,username,email,password_hash,role) VALUES(1,'one','one@x','hash','user'),(2,'two','two@x','hash','user')")
	if err != nil {
		t.Fatal(err)
	}
	repo := repository.NewSQLiteQuestionRepository(db)
	makeQ := func(uid int64) model.WrongQuestion {
		q, e := repo.Create(model.WrongQuestion{UserID: uid, Subject: "考研数学", SubjectID: "math_grad", ClassificationStatus: "confirmed", QuestionCore: "question", SemanticSummary: "summary", MasteryStatus: "unmastered", SourceType: "manual", CreatedAt: time.Now(), UpdatedAt: time.Now()})
		if e != nil {
			t.Fatal(e)
		}
		return q
	}
	base := makeQ(1)
	same := makeQ(1)
	foreign := makeQ(2)
	broken := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if broken {
			w.WriteHeader(503)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"embedding": []float64{1, 0, 0}}}})
	}))
	defer server.Close()
	cfg := config.EmbeddingModelConfig{ProviderType: "openai_compatible", BaseURL: server.URL, Model: "test", APIKey: "test"}
	local := &LocalVector{DB: db, Config: func() config.EmbeddingModelConfig { return cfg }}
	local.step(context.Background())
	local.step(context.Background())
	local.step(context.Background())
	got, err := local.Search(base, "semantic", 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].QuestionID != same.ID || got[0].QuestionID == foreign.ID {
		t.Fatalf("cross-user vector result: %v", got)
	}
	cfg.Model = "changed"
	got, err = local.Search(base, "semantic", 10, false)
	if err != nil || len(got) != 0 {
		t.Fatal("mixed embedding models")
	}
	cfg.Model = "test"
	same.QuestionCore = "updated"
	same.SemanticSummary = "updated"
	same.UpdatedAt = time.Now()
	if _, err = repo.Update(same); err != nil {
		t.Fatal(err)
	}
	var n int
	db.QueryRow("SELECT count(*) FROM local_vector WHERE question_id=?", same.ID).Scan(&n)
	if n != 0 {
		t.Fatal("stale vector survived update")
	}
	broken = true
	local.step(context.Background())
	var status string
	db.QueryRow("SELECT status FROM vector_job WHERE question_id=?", same.ID).Scan(&status)
	if status != "failed" {
		t.Fatal(status)
	}
	broken = false
	db.Exec("UPDATE vector_job SET next_attempt=0")
	local.step(context.Background())
	db.QueryRow("SELECT status FROM vector_job WHERE question_id=?", same.ID).Scan(&status)
	if status != "done" {
		t.Fatal(status)
	}
}
func TestCosineRejectsWrongDimensions(t *testing.T) {
	if cosine([]float64{1}, vectorBytes([]float64{1, 2})) != -2 {
		t.Fatal("dimension mismatch accepted")
	}
}
