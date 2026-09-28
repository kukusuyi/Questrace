package service

import (
	"fmt"
	"github.com/kukusuyi/Questrace/backend/internal/infra/sqlite"
	"github.com/kukusuyi/Questrace/backend/internal/repository"
	"testing"
	"time"
)

func TestReviewLifecycle(t *testing.T) {
	db, err := sqlite.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`INSERT INTO user(id,username)VALUES(1,'one'),(2,'two'); INSERT INTO wrong_question(user_id,subject,question_core,semantic_summary)VALUES(1,'math','cos x',''),(2,'math','private','')`)
	if err != nil {
		t.Fatal(err)
	}
	now := int64(2000000000)
	svc := ReviewService{DB: db, Now: func() time.Time { return time.Unix(now, 0) }}
	for i := 1; i <= 5; i++ {
		sess, err := svc.Create(1, ReviewCreate{})
		if err != nil || len(sess.Items) != 1 {
			t.Fatalf("create: %+v %v", sess, err)
		}
		if _, err = svc.Get(2, sess.ID); err == nil {
			t.Fatal("foreign session visible")
		}
		in := ReviewSubmit{QuestionID: sess.Items[0].QuestionID, SubmissionID: fmt.Sprint(i), Result: "correct"}
		result, err := svc.Submit(1, sess.ID, in)
		if err != nil {
			t.Fatal(err)
		}
		again, err := svc.Submit(1, sess.ID, in)
		if err != nil || again != result {
			t.Fatal("retry was not idempotent", again, err)
		}
		conflicting := in
		conflicting.Result = "forgot"
		if _, err = svc.Submit(1, sess.ID, conflicting); err == nil {
			t.Fatal("idempotency key accepted a changed payload")
		}
		expected := "learning"
		if i >= 3 {
			expected = "mastered"
		}
		if result.Mastery != expected {
			t.Fatal(result)
		}
		if i == 1 {
			early, err := svc.Create(1, ReviewCreate{})
			if err != nil {
				t.Fatal(err)
			}
			got, err := svc.Submit(1, early.ID, ReviewSubmit{QuestionID: in.QuestionID, SubmissionID: "early", Result: "correct"})
			if err != nil || got != result {
				t.Fatalf("early repetition advanced: %+v %v", got, err)
			}
		}
		now = result.DueAt
	}
	sess, err := svc.Create(1, ReviewCreate{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := svc.Submit(1, sess.ID, ReviewSubmit{QuestionID: 1, SubmissionID: "forgot", Result: "forgot"})
	if err != nil || result.Mastery != "unmastered" {
		t.Fatal(result, err)
	}
	var n int
	db.QueryRow(`SELECT count(*) FROM review_record`).Scan(&n)
	if n != 7 {
		t.Fatalf("duplicate records %d", n)
	}
	// Manual changes reset scheduling, content-only edits do not.
	repo := repository.NewSQLiteQuestionRepository(db)
	q, ok := repo.GetByID(1)
	if !ok {
		t.Fatal("missing question")
	}
	q.MasteryStatus = "mastered"
	if q, err = repo.Update(q); err != nil {
		t.Fatal(err)
	}
	var streak int
	var due int64
	db.QueryRow(`SELECT streak,due_at FROM review_plan WHERE question_id=1`).Scan(&streak, &due)
	if streak != 0 || due < time.Now().Add(6*24*time.Hour).Unix() {
		t.Fatal("manual reset missing")
	}
	q.QuestionCore = "new"
	if q, err = repo.Update(q); err != nil {
		t.Fatal(err)
	}
	var after int64
	db.QueryRow(`SELECT due_at FROM review_plan WHERE question_id=1`).Scan(&after)
	if after != due {
		t.Fatal("content edit reset plan")
	}
}
func TestAdvance(t *testing.T) {
	m, n, d := Advance("partial", "mastered", 3, 50, 10, 100)
	if m != "learning" || n != 0 || d != 86500 {
		t.Fatal(m, n, d)
	}
	m, n, d = Advance("correct", "learning", 1, 200, 50, 100)
	if m != "learning" || n != 1 || d != 200 {
		t.Fatal("early success advanced")
	}
}
