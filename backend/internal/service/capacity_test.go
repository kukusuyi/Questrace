package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/kukusuyi/Questrace/backend/internal/config"
	"github.com/kukusuyi/Questrace/backend/internal/domain/model"
	"github.com/kukusuyi/Questrace/backend/internal/infra/sqlite"
	"github.com/kukusuyi/Questrace/backend/internal/repository"
)

// Opt-in: creates a 100k-question / 1024-dimension dataset in a temporary directory.
// Embedding HTTP is a local stub; timings measure persistence/filtering/scoring.
func TestCapacity(t *testing.T) {
	if os.Getenv("QUESTRACE_CAPACITY_TEST") != "1" {
		t.Skip("set QUESTRACE_CAPACITY_TEST=1 for the 100k-question capacity test")
	}
	db, err := sqlite.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	values := make([]float64, 1024)
	for i := range values {
		values[i] = float64(i%7+1) / 10
	}
	blob := vectorBytes(values)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"embedding": values}}})
	}))
	defer provider.Close()
	cfg := config.EmbeddingModelConfig{ProviderType: "openai_compatible", BaseURL: provider.URL, Model: "capacity", APIKey: "mock"}
	local := &LocalVector{DB: db, Config: func() config.EmbeddingModelConfig { return cfg }}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for i := 1; i <= 20; i++ {
		if _, err = tx.Exec("INSERT INTO user(id,username,email,password_hash,role) VALUES(?,?,?,'hash','user')", i, fmt.Sprintf("user%d", i), fmt.Sprintf("user%d@local", i)); err != nil {
			t.Fatal(err)
		}
	}
	q, err := tx.Prepare("INSERT INTO wrong_question(id,user_id,subject,subject_id,classification_status,question_core,semantic_summary) VALUES(?,?,'考研数学','math_grad','confirmed','question','summary')")
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()
	v, err := tx.Prepare("INSERT INTO local_vector(question_id,vector_type,model,dimension,content_hash,vector,format_version) VALUES(?,'semantic',?,1024,'test',?,2)")
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	for i := 1; i <= 100000; i++ {
		if _, err = q.Exec(i, (i-1)%20+1); err != nil {
			t.Fatal(err)
		}
		if _, err = v.Exec(i, local.modelKey(cfg), blob); err != nil {
			t.Fatal(err)
		}
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var timings []time.Duration
	var wg sync.WaitGroup
	repo := repository.NewSQLiteQuestionRepository(db)
	for uid := int64(1); uid <= 20; uid++ {
		wg.Add(1)
		go func(uid int64) {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				start := time.Now()
				items, n, e := repo.List(repository.QuestionFilter{UserID: uid, Page: 1, PageSize: 20})
				if e != nil || n != 5000 || len(items) != 20 {
					t.Errorf("list: %d %d %v", n, len(items), e)
					return
				}
				mu.Lock()
				timings = append(timings, time.Since(start))
				mu.Unlock()
			}
		}(uid)
	}
	wg.Wait()
	sort.Slice(timings, func(i, j int) bool { return timings[i] < timings[j] })
	if len(timings) != 200 {
		t.Fatal("incomplete concurrent requests")
	}
	p95 := timings[len(timings)*95/100]
	start := time.Now()
	found, err := local.Search(model.WrongQuestion{UserID: 1, Subject: "考研数学", SubjectID: "math_grad", ClassificationStatus: "confirmed", QuestionCore: "q"}, "semantic", 10, false)
	search := time.Since(start)
	if err != nil || len(found) != 10 {
		t.Fatalf("search: %v %v", found, err)
	}
	for _, item := range found {
		if (item.QuestionID-1)%20 != 0 {
			t.Fatal("cross-user search")
		}
	}
	t.Logf("100k questions / 20 users / 1024 dimensions: 20 concurrent list clients P95=%v, user-filtered search=%v", p95, search)
	if p95 > 500*time.Millisecond {
		t.Errorf("list P95 exceeds 500 ms: %v", p95)
	}
	if search > 2*time.Second {
		t.Errorf("local search exceeds 2 s: %v", search)
	}
}
