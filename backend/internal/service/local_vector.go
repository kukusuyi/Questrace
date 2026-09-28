package service

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/kukusuyi/Questrace/backend/internal/config"
	"github.com/kukusuyi/Questrace/backend/internal/domain/model"
	ai "github.com/kukusuyi/Questrace/backend/internal/infra/ai"
	apperrors "github.com/kukusuyi/Questrace/backend/internal/pkg/errors"
	"github.com/kukusuyi/Questrace/backend/internal/repository"
)

type LocalVector struct {
	DB     *sql.DB
	Config func() config.EmbeddingModelConfig
}

func (s *LocalVector) modelKey(c config.EmbeddingModelConfig) string {
	return c.ProviderType + "|" + strings.TrimRight(c.BaseURL, "/") + "|" + c.Model
}
func vectorBytes(v []float64) []byte {
	b := make([]byte, len(v)*4)
	for i, x := range v {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(float32(x)))
	}
	return b
}
func cosine(v []float64, b []byte) float64 {
	if len(b) != len(v)*4 {
		return -2
	}
	var dot, a, c float64
	for i, x := range v {
		y := float64(math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:])))
		dot += x * y
		a += x * x
		c += y * y
	}
	if a == 0 || c == 0 {
		return -2
	}
	return dot / math.Sqrt(a*c)
}
func (s *LocalVector) Run(ctx context.Context) {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			s.step(ctx)
		}
	}
}
func (s *LocalVector) step(ctx context.Context) {
	cfg := s.Config()
	if cfg.APIKey == "" {
		return
	}
	var id, revision int64
	err := s.DB.QueryRowContext(ctx, "SELECT question_id,revision FROM vector_job WHERE status <> 'done' AND next_attempt<=? ORDER BY next_attempt,question_id LIMIT 1", time.Now().Unix()).Scan(&id, &revision)
	if err != nil {
		return
	}
	client, err := ai.NewEmbeddingClient(cfg)
	if err != nil {
		return
	}
	q, ok := repository.NewSQLiteQuestionRepository(s.DB).GetByID(id)
	if !ok {
		return
	}
	type item struct {
		kind, hash string
		vector     []float64
	}
	var items []item
	if !q.IsDeleted && q.ClassificationStatus == "confirmed" && q.SubjectID != "" {
		for _, kind := range []string{"semantic", "mistake"} {
			text := buildSearchText(q, kind)
			if text == "" {
				continue
			}
			var v []float64
			v, err = client.Embed(ctx, text)
			if err != nil {
				break
			}
			if len(v) == 0 {
				err = fmt.Errorf("empty embedding")
				break
			}
			items = append(items, item{kind, hashText(text), v})
		}
	}
	if err != nil {
		_, _ = s.DB.ExecContext(ctx, "UPDATE vector_job SET status='failed',attempts=attempts+1,next_attempt=?,error='模型请求失败，可检查设置后重试' WHERE question_id=? AND revision=?", time.Now().Add(time.Minute).Unix(), id, revision)
		return
	}
	if s.modelKey(s.Config()) != s.modelKey(cfg) {
		return
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	var current int64
	if err = tx.QueryRow("SELECT revision FROM vector_job WHERE question_id=?", id).Scan(&current); err != nil || current != revision {
		return
	}
	if _, err = tx.Exec("DELETE FROM local_vector WHERE question_id=?", id); err != nil {
		return
	}
	for _, item := range items {
		if _, err = tx.Exec("INSERT INTO local_vector(question_id,vector_type,model,dimension,content_hash,vector,format_version) VALUES(?,?,?,?,?,?,?)", id, item.kind, s.modelKey(cfg), len(item.vector), item.hash, vectorBytes(item.vector), vectorFormatVersion); err != nil {
			return
		}
	}
	if _, err = tx.Exec("UPDATE vector_job SET status='done',error='' WHERE question_id=? AND revision=?", id, revision); err != nil {
		return
	}
	_ = tx.Commit()
}
func (s *LocalVector) Search(base model.WrongQuestion, kind string, limit int, filter bool) ([]SimilarSearchItem, error) {
	if base.SubjectID == "" || base.ClassificationStatus != "confirmed" {
		return nil, apperrors.New(409, 40902, "请先确认题目学科，再查找相似题")
	}
	if base.RecallScope != "" && base.RecallScope != "course" && base.RecallScope != "subject" {
		return nil, apperrors.New(400, 40001, "召回范围无效")
	}
	if base.SubjectID == "cs408" && base.RecallScope != "subject" && base.CourseID == "" {
		return nil, apperrors.New(409, 40902, "请选择408课程，或选择整个408范围")
	}
	if buildSearchText(base, kind) == "" {
		return []SimilarSearchItem{}, nil
	}
	cfg := s.Config()
	if cfg.APIKey == "" {
		return nil, apperrors.New(503, 50301, "请管理员先配置 Embedding 模型")
	}
	client, err := ai.NewEmbeddingClient(cfg)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	v, err := client.Embed(ctx, buildSearchText(base, kind))
	if err != nil {
		return nil, apperrors.New(503, 50302, "向量模型暂不可用")
	}
	if base.ID > 0 {
		var revision int64
		var status, subject, course string
		err = s.DB.QueryRowContext(ctx, "SELECT revision,classification_status,subject_id,course_id FROM wrong_question WHERE id=? AND user_id=? AND is_deleted=0", base.ID, base.UserID).Scan(&revision, &status, &subject, &course)
		if err != nil || revision != base.Revision || status != "confirmed" || subject != base.SubjectID || course != base.CourseID {
			return nil, apperrors.New(409, 40901, "题目已变更，请刷新后重新检索")
		}
	}
	query := `SELECT v.question_id,v.vector FROM local_vector v JOIN wrong_question q ON q.id=v.question_id WHERE q.user_id=? AND q.is_deleted=0 AND v.vector_type=? AND v.model=? AND v.dimension=? AND q.id<>? AND q.classification_status='confirmed' AND q.subject_id=? AND v.format_version=?`
	args := []any{base.UserID, kind, s.modelKey(cfg), len(v), base.ID, base.SubjectID, vectorFormatVersion}
	if base.SubjectID == "cs408" && base.RecallScope != "subject" {
		query += " AND q.course_id=?"
		args = append(args, base.CourseID)
	}
	if filter && base.AnalysisStale {
		return []SimilarSearchItem{}, nil
	}
	if filter {
		query += " AND q.analysis_stale=0"
		groups := []struct {
			kind   string
			values []string
		}{{"knowledge_point", base.Tags.KnowledgePoints}, {"problem_type", base.Tags.ProblemType}, {"method", base.Tags.Method}}
		if kind == "mistake" {
			groups = append(groups, struct {
				kind   string
				values []string
			}{"mistake_reason", base.Tags.MistakeReason})
		}
		for _, g := range groups {
			if len(g.values) == 0 {
				continue
			}
			b, _ := json.Marshal(g.values)
			query += ` AND EXISTS(SELECT 1 FROM wrong_question_tag qt JOIN tag t ON t.id=qt.tag_id WHERE qt.question_id=q.id AND t.is_active=1 AND t.tag_type=? AND t.tag_name IN (SELECT value FROM json_each(?)))`
			args = append(args, g.kind, string(b))
		}
	}
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	items := make([]SimilarSearchItem, 0, limit+1)
	for rows.Next() {
		var id int64
		var b []byte
		if err = rows.Scan(&id, &b); err != nil {
			return nil, err
		}
		score := cosine(v, b)
		if math.IsNaN(score) || math.IsInf(score, 0) || score < -1 {
			continue
		}
		items = append(items, SimilarSearchItem{QuestionID: id, Score: score})
		sort.Slice(items, func(i, j int) bool {
			if items[i].Score == items[j].Score {
				return items[i].QuestionID < items[j].QuestionID
			}
			return items[i].Score > items[j].Score
		})
		if len(items) > limit {
			items = items[:limit]
		}
	}
	return items, rows.Err()
}

// IndexPending distinguishes an empty candidate set from pending/failed indexing.
func (s *LocalVector) IndexPending(base model.WrongQuestion) bool {
	if s == nil || s.DB == nil {
		return false
	}
	query := `SELECT count(*) FROM vector_job j JOIN wrong_question q ON q.id=j.question_id WHERE q.user_id=? AND q.subject_id=? AND q.classification_status='confirmed' AND q.is_deleted=0 AND j.status<>'done'`
	args := []any{base.UserID, base.SubjectID}
	if base.SubjectID == "cs408" && base.RecallScope != "subject" {
		query += " AND q.course_id=?"
		args = append(args, base.CourseID)
	}
	var n int
	return s.DB.QueryRow(query, args...).Scan(&n) == nil && n > 0
}
