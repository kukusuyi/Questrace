package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/kukusuyi/Questrace/backend/internal/domain/enum"
	"github.com/kukusuyi/Questrace/backend/internal/domain/model"
)

type rowScanner interface {
	Scan(dest ...any) error
}

type typedTagRecord struct {
	TagType string
	Name    string
}

func collectTypedTagRecords(groups model.TagCollections) []typedTagRecord {
	items := make([]typedTagRecord, 0, len(groups.KnowledgePoints)+len(groups.ProblemType)+len(groups.Method)+len(groups.MistakeReason))
	for _, name := range groups.KnowledgePoints {
		items = append(items, typedTagRecord{TagType: string(enum.TagTypeKnowledgePoint), Name: name})
	}
	for _, name := range groups.ProblemType {
		items = append(items, typedTagRecord{TagType: string(enum.TagTypeProblemType), Name: name})
	}
	for _, name := range groups.Method {
		items = append(items, typedTagRecord{TagType: string(enum.TagTypeMethod), Name: name})
	}
	for _, name := range groups.MistakeReason {
		items = append(items, typedTagRecord{TagType: string(enum.TagTypeMistakeReason), Name: name})
	}

	return items
}

func buildInClause(values []string) string {
	if len(values) == 0 {
		return ""
	}

	parts := make([]string, 0, len(values))
	for range values {
		parts = append(parts, "?")
	}

	return strings.Join(parts, ", ")
}

func buildInt64InClause(values []int64) string {
	if len(values) == 0 {
		return ""
	}

	parts := make([]string, 0, len(values))
	for range values {
		parts = append(parts, "?")
	}

	return strings.Join(parts, ", ")
}

func nullableInt64Value(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableStringValue(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullableTimeValue(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func scanTag(scanner rowScanner) (model.Tag, error) {
	var item model.Tag
	var deletedAt sql.NullTime
	err := scanner.Scan(
		&item.ID,
		&item.UserID,
		&item.TagName,
		&item.TagType,
		&item.UsageCount,
		&item.IsActive,
		&deletedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return model.Tag{}, err
	}
	if deletedAt.Valid {
		value := deletedAt.Time
		item.DeletedAt = &value
	}

	return item, nil
}

func scanFileRecord(scanner rowScanner) (model.FileRecord, error) {
	var item model.FileRecord
	var questionID sql.NullInt64
	err := scanner.Scan(
		&item.ID,
		&item.UserID,
		&questionID,
		&item.StorageProvider,
		&item.BucketName,
		&item.ObjectKey,
		&item.FileName,
		&item.FileURL,
		&item.FileSize,
		&item.MIMEType,
		&item.FileType,
		&item.CreatedAt,
	)
	if err != nil {
		return model.FileRecord{}, err
	}
	if questionID.Valid {
		value := questionID.Int64
		item.QuestionID = &value
	}

	return item, nil
}

func scanQuestion(scanner rowScanner) (model.WrongQuestion, error) {
	var item model.WrongQuestion
	var sourceImageID sql.NullInt64
	var chapter sql.NullString
	var standardSolution sql.NullString
	var wrongSolution sql.NullString
	var mistakeSummary sql.NullString
	var difficultyLevel sql.NullInt64
	var sourceImageURL sql.NullString
	var deletedAt sql.NullTime
	err := scanner.Scan(
		&item.ID,
		&item.UserID,
		&item.SubjectID, &item.CourseID, &item.ClassificationStatus, &item.AnalysisStale, &item.Revision,
		&item.Subject,
		&chapter,
		&item.QuestionCore,
		&standardSolution,
		&wrongSolution,
		&item.SemanticSummary,
		&mistakeSummary,
		&difficultyLevel,
		&item.MasteryStatus,
		&item.SourceType,
		&sourceImageID,
		&sourceImageURL,
		&item.IsDeleted,
		&deletedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return model.WrongQuestion{}, err
	}
	item.Chapter = chapter.String
	item.StandardSolution = standardSolution.String
	item.WrongSolution = wrongSolution.String
	item.MistakeSummary = mistakeSummary.String
	item.SourceImageURL = sourceImageURL.String
	if difficultyLevel.Valid {
		item.DifficultyLevel = int(difficultyLevel.Int64)
	}
	if sourceImageID.Valid {
		value := sourceImageID.Int64
		item.SourceImageID = &value
	}
	if deletedAt.Valid {
		value := deletedAt.Time
		item.DeletedAt = &value
	}

	return item, nil
}

func formatMissingTagError(tagType, tagName string) error {
	return fmt.Errorf("tag not found or inactive: type=%s name=%s", tagType, tagName)
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}
