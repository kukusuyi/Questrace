package service

import (
	"database/sql"
	"github.com/kukusuyi/Questrace/backend/internal/domain/model"
	"github.com/kukusuyi/Questrace/backend/internal/repository"
	"strings"
	"time"

	apperrors "github.com/kukusuyi/Questrace/backend/internal/pkg/errors"
)

type ReviewService struct {
	DB  *sql.DB
	Now func() time.Time
}

func (s ReviewService) now() int64 {
	if s.Now != nil {
		return s.Now().Unix()
	}
	return time.Now().Unix()
}

type ReviewCreate struct {
	SubjectID     string  `json:"subject_id"`
	CourseID      string  `json:"course_id"`
	Chapter       string  `json:"chapter"`
	Subject       string  `json:"subject"`
	TagIDs        []int64 `json:"tag_ids"`
	MasteryStatus string  `json:"mastery_status"`
	Count         int     `json:"count"`
}
type ReviewSession struct {
	ID             int64        `json:"id"`
	RequestedCount int          `json:"requested_count"`
	CreatedAt      int64        `json:"created_at"`
	Items          []ReviewItem `json:"items"`
}
type ReviewItem struct {
	SubjectID            string `json:"subject_id"`
	Subject              string `json:"subject"`
	CourseID             string `json:"course_id"`
	Chapter              string `json:"chapter"`
	ClassificationStatus string `json:"classification_status"`
	AnalysisStale        bool   `json:"analysis_stale"`
	QuestionID           int64  `json:"question_id"`
	QuestionCore         string `json:"question_core"`
	StandardSolution     string `json:"standard_solution"`
	SourceImageURL       string `json:"source_image_url"`
	Result               string `json:"result"`
	Deleted              bool   `json:"deleted"`
}
type ReviewSummary struct {
	Due      int                    `json:"due"`
	Sessions []ReviewSessionSummary `json:"sessions"`
}
type ReviewSessionSummary struct {
	ID        int64 `json:"id"`
	CreatedAt int64 `json:"created_at"`
	Total     int   `json:"total"`
	Done      int   `json:"done"`
}
type ReviewSubmit struct {
	QuestionID   int64  `json:"question_id"`
	SubmissionID string `json:"submission_id"`
	Result       string `json:"result"`
	Note         string `json:"note"`
}
type ReviewOutcome struct {
	Mastery string `json:"mastery_status"`
	DueAt   int64  `json:"due_at"`
}

func (s ReviewService) Summary(uid int64) (ReviewSummary, error) {
	out := ReviewSummary{Sessions: []ReviewSessionSummary{}}
	err := s.DB.QueryRow(`SELECT count(*) FROM review_plan p JOIN wrong_question q ON q.id=p.question_id WHERE q.user_id=? AND q.is_deleted=0 AND p.due_at<=?`, uid, s.now()).Scan(&out.Due)
	if err != nil {
		return out, err
	}
	rows, err := s.DB.Query(`SELECT s.id,s.created_at,count(i.question_id),coalesce(sum(CASE WHEN i.result<>'' OR q.is_deleted=1 THEN 1 ELSE 0 END),0) FROM review_session s JOIN review_session_item i ON i.session_id=s.id JOIN wrong_question q ON q.id=i.question_id WHERE s.user_id=? GROUP BY s.id ORDER BY s.id DESC LIMIT 100`, uid)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var x ReviewSessionSummary
		if err = rows.Scan(&x.ID, &x.CreatedAt, &x.Total, &x.Done); err != nil {
			return out, err
		}
		out.Sessions = append(out.Sessions, x)
	}
	return out, rows.Err()
}
func (s ReviewService) Create(uid int64, in ReviewCreate) (ReviewSession, error) {
	if in.Count == 0 {
		in.Count = 10
	}
	if in.Count < 1 || in.Count > 100 {
		return ReviewSession{}, apperrors.New(400, 40000, "题数应为 1–100")
	}
	if in.MasteryStatus != "" && in.MasteryStatus != "unmastered" && in.MasteryStatus != "learning" && in.MasteryStatus != "mastered" {
		return ReviewSession{}, apperrors.New(400, 40000, "掌握状态无效")
	}
	where := `q.user_id=? AND q.is_deleted=0`
	args := []any{uid}
	for _, f := range []struct{ column, value string }{{"subject_id", in.SubjectID}, {"course_id", in.CourseID}, {"chapter", in.Chapter}} {
		if f.value != "" {
			where += " AND q." + f.column + "=?"
			args = append(args, f.value)
		}
	}
	if in.Subject != "" {
		where += ` AND q.subject=?`
		args = append(args, in.Subject)
	}
	if in.MasteryStatus != "" {
		where += ` AND q.mastery_status=?`
		args = append(args, in.MasteryStatus)
	} else {
		where += ` AND (p.due_at<=? OR q.mastery_status<>'mastered')`
		args = append(args, s.now())
	}
	if len(in.TagIDs) > 0 {
		where += " AND q.analysis_stale=0"
		where += ` AND EXISTS(SELECT 1 FROM wrong_question_tag w JOIN tag t ON t.id=w.tag_id WHERE w.question_id=q.id AND t.is_active=1 AND t.id IN (` + strings.TrimRight(strings.Repeat("?,", len(in.TagIDs)), ",") + `))`
		for _, id := range in.TagIDs {
			args = append(args, id)
		}
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return ReviewSession{}, err
	}
	defer tx.Rollback()
	args = append(args, s.now(), in.Count)
	rows, err := tx.Query(`SELECT q.id FROM wrong_question q JOIN review_plan p ON p.question_id=q.id WHERE `+where+` ORDER BY CASE WHEN p.due_at<=? THEN 0 WHEN q.mastery_status='unmastered' THEN 1 ELSE 2 END,p.due_at,q.id LIMIT ?`, args...)
	if err != nil {
		return ReviewSession{}, err
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return ReviewSession{}, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return ReviewSession{}, err
	}
	if len(ids) == 0 {
		return ReviewSession{}, apperrors.New(400, 40000, "没有符合条件的题目")
	}
	res, err := tx.Exec(`INSERT INTO review_session(user_id,requested_count,created_at) VALUES(?,?,?)`, uid, in.Count, s.now())
	if err != nil {
		return ReviewSession{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return ReviewSession{}, err
	}
	for pos, qid := range ids {
		if _, err = tx.Exec(`INSERT INTO review_session_item(session_id,question_id,position) VALUES(?,?,?)`, id, qid, pos); err != nil {
			return ReviewSession{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return ReviewSession{}, err
	}
	return s.Get(uid, id)
}
func (s ReviewService) Get(uid, id int64) (ReviewSession, error) {
	out := ReviewSession{Items: []ReviewItem{}}
	err := s.DB.QueryRow(`SELECT id,requested_count,created_at FROM review_session WHERE id=? AND user_id=?`, id, uid).Scan(&out.ID, &out.RequestedCount, &out.CreatedAt)
	if err == sql.ErrNoRows {
		return out, apperrors.New(404, 40400, "练习不存在")
	}
	if err != nil {
		return out, err
	}
	rows, err := s.DB.Query(`SELECT q.id,q.question_core,coalesce(q.standard_solution,''),coalesce(q.source_image_url,''),i.result,q.is_deleted,q.subject_id,q.subject,q.course_id,coalesce(q.chapter,''),q.classification_status,q.analysis_stale FROM review_session_item i JOIN wrong_question q ON q.id=i.question_id WHERE i.session_id=? AND q.user_id=? ORDER BY i.position`, id, uid)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var item ReviewItem
		if err = rows.Scan(&item.QuestionID, &item.QuestionCore, &item.StandardSolution, &item.SourceImageURL, &item.Result, &item.Deleted, &item.SubjectID, &item.Subject, &item.CourseID, &item.Chapter, &item.ClassificationStatus, &item.AnalysisStale); err != nil {
			return out, err
		}
		out.Items = append(out.Items, item)
	}
	return out, rows.Err()
}

// Advance counts only due repetitions; early successes never accelerate mastery.
func Advance(result, mastery string, streak int, due, last, now int64) (string, int, int64) {
	if result == "forgot" {
		return "unmastered", 0, now + 86400
	}
	if result == "partial" {
		return "learning", 0, now + 86400
	}
	if last != 0 && due > now {
		return mastery, streak, due
	}
	streak++
	days := 30
	switch streak {
	case 1:
		days = 1
	case 2:
		days = 3
	case 3:
		days = 7
	case 4:
		days = 14
	}
	if streak >= 3 {
		mastery = "mastered"
	} else if mastery != "mastered" {
		mastery = "learning"
	}
	return mastery, streak, now + int64(days)*86400
}
func (s ReviewService) Submit(uid, id int64, in ReviewSubmit) (ReviewOutcome, error) {
	out := ReviewOutcome{}
	if in.SubmissionID == "" || len(in.SubmissionID) > 128 || len(in.Note) > 10000 || (in.Result != "forgot" && in.Result != "partial" && in.Result != "correct") {
		return out, apperrors.New(400, 40000, "复习结果或提交标识无效")
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	var oldResult, mastery string
	var deleted bool
	err = tx.QueryRow(`SELECT i.result,q.mastery_status,q.is_deleted FROM review_session_item i JOIN review_session s ON s.id=i.session_id JOIN wrong_question q ON q.id=i.question_id WHERE s.id=? AND s.user_id=? AND q.user_id=? AND q.id=?`, id, uid, uid, in.QuestionID).Scan(&oldResult, &mastery, &deleted)
	if err == sql.ErrNoRows {
		return out, apperrors.New(404, 40400, "练习题目不存在")
	}
	if err != nil {
		return out, err
	}
	if deleted {
		return out, apperrors.New(410, 41000, "题目已删除")
	}
	var priorQ, priorSession int64
	var priorResult, priorNote string
	err = tx.QueryRow(`SELECT question_id,session_id,mastery_after,next_due_at,review_result,coalesce(note,'') FROM review_record WHERE user_id=? AND submission_id=?`, uid, in.SubmissionID).Scan(&priorQ, &priorSession, &out.Mastery, &out.DueAt, &priorResult, &priorNote)
	if err == nil {
		if priorQ != in.QuestionID || priorSession != id || priorResult != in.Result || priorNote != in.Note {
			return out, apperrors.New(409, 40900, "提交标识已被使用")
		}
		return out, nil
	}
	if err != sql.ErrNoRows {
		return out, err
	}
	if oldResult != "" {
		return out, apperrors.New(409, 40900, "此题已提交，请刷新练习")
	}
	var streak int
	var due, last int64
	if err = tx.QueryRow(`SELECT streak,due_at,last_reviewed_at FROM review_plan WHERE question_id=?`, in.QuestionID).Scan(&streak, &due, &last); err != nil {
		return out, err
	}
	now := s.now()
	out.Mastery, streak, out.DueAt = Advance(in.Result, mastery, streak, due, last, now)
	_, err = tx.Exec(`INSERT INTO review_record(user_id,question_id,review_result,mastery_before,mastery_after,note,reviewed_at,session_id,submission_id,next_due_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, uid, in.QuestionID, in.Result, mastery, out.Mastery, in.Note, time.Unix(now, 0), id, in.SubmissionID, out.DueAt)
	if err != nil {
		return out, err
	}
	statements := []struct {
		q string
		a []any
	}{
		{`UPDATE review_plan SET streak=?,due_at=?,last_reviewed_at=? WHERE question_id=?`, []any{streak, out.DueAt, now, in.QuestionID}},
		{`UPDATE wrong_question SET mastery_status=?,updated_at=? WHERE id=?`, []any{out.Mastery, time.Unix(now, 0), in.QuestionID}},
		{`UPDATE review_session_item SET result=? WHERE session_id=? AND question_id=?`, []any{in.Result, id, in.QuestionID}},
	}
	for _, st := range statements {
		if _, err = tx.Exec(st.q, st.a...); err != nil {
			return out, err
		}
	}
	return out, tx.Commit()
}
func (s ReviewService) History(uid int64) ([]model.ReviewRecord, error) {
	return (repository.ReviewRepository{DB: s.DB}).History(uid)
}
