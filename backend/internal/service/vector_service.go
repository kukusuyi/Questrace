package service

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/kukusuyi/Questrace/backend/internal/domain/enum"
	"github.com/kukusuyi/Questrace/backend/internal/domain/model"
)

type VectorService struct{ Local *LocalVector }
type SimilarSearchItem struct {
	QuestionID int64
	Score      float64
}

// Triggers enqueue vector work in the same transaction as a question write.
func (s *VectorService) Upsert(question model.WrongQuestion) error { return nil }
func (s *VectorService) Delete(id int64) error {
	_, err := s.Local.DB.Exec("DELETE FROM local_vector WHERE question_id=?", id)
	return err
}
func (s *VectorService) Search(base model.WrongQuestion, kind string, limit int, filter bool) ([]SimilarSearchItem, error) {
	return s.Local.Search(base, kind, limit, filter)
}

const vectorFormatVersion = 2

func buildSearchText(base model.WrongQuestion, vectorType string) string {
	var body string
	if vectorType == string(enum.VectorTypeMistake) {
		if strings.TrimSpace(base.WrongSolution) == "" {
			return ""
		}
		body = base.WrongSolution
		if !base.AnalysisStale && strings.TrimSpace(base.MistakeSummary) != "" {
			body = base.MistakeSummary
		}
	} else {
		body = base.QuestionCore
		if !base.AnalysisStale && strings.TrimSpace(base.SemanticSummary) != "" {
			body = base.SemanticSummary
		}
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	return "学科：" + base.Subject + " [" + base.SubjectID + "]\n课程：" + base.CourseID + "\n章节：" + base.Chapter + "\n" + body
}

func hashText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
