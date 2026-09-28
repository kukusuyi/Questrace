package repository

import (
	"github.com/kukusuyi/Questrace/backend/internal/pkg/searchtext"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kukusuyi/Questrace/backend/internal/domain/model"
)

type QuestionFilter struct {
	SubjectID            string
	CourseID             string
	ClassificationStatus string
	UserID               int64
	Page                 int
	PageSize             int
	Subject              string
	Chapter              string
	Keyword              string
	TagNames             []string
	TagIDs               []int64
	MasteryStatus        string
	DifficultyLevel      int
	SourceType           string
}

type QuestionDashboardMetrics struct {
	TotalQuestions   int
	TodayAdded       int
	UnmasteredCount  int
	ImageBoundCount  int
	MasteryStatusMap map[string]int
	SourceTypeMap    map[string]int
}

type QuestionRepository interface {
	Create(question model.WrongQuestion) (model.WrongQuestion, error)
	Update(question model.WrongQuestion) (model.WrongQuestion, error)
	GetByID(id int64) (model.WrongQuestion, bool)
	List(filter QuestionFilter) ([]model.WrongQuestion, int, error)
	DashboardMetrics(userID int64, now time.Time) (QuestionDashboardMetrics, error)
	SoftDelete(id int64, deletedAt time.Time) (model.WrongQuestion, bool, error)
}

type InMemoryQuestionRepository struct {
	mu     sync.RWMutex
	nextID int64
	items  map[int64]*model.WrongQuestion
}

func NewInMemoryQuestionRepository() *InMemoryQuestionRepository {
	return &InMemoryQuestionRepository{
		nextID: 1,
		items:  make(map[int64]*model.WrongQuestion),
	}
}

func (r *InMemoryQuestionRepository) Create(question model.WrongQuestion) (model.WrongQuestion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	question.ID = r.nextID
	r.nextID++
	copyValue := cloneQuestion(question)
	r.items[question.ID] = &copyValue

	return cloneQuestion(copyValue), nil
}

func (r *InMemoryQuestionRepository) Update(question model.WrongQuestion) (model.WrongQuestion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	copyValue := cloneQuestion(question)
	r.items[question.ID] = &copyValue
	return cloneQuestion(copyValue), nil
}

func (r *InMemoryQuestionRepository) GetByID(id int64) (model.WrongQuestion, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, ok := r.items[id]
	if !ok {
		return model.WrongQuestion{}, false
	}

	return cloneQuestion(*item), true
}

func (r *InMemoryQuestionRepository) List(filter QuestionFilter) ([]model.WrongQuestion, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make([]model.WrongQuestion, 0)
	for _, item := range r.items {
		if item.IsDeleted {
			continue
		}

		if filter.UserID > 0 && item.UserID != filter.UserID {
			continue
		}

		if filter.SubjectID != "" && item.SubjectID != filter.SubjectID {
			continue
		}
		if filter.CourseID != "" && item.CourseID != filter.CourseID {
			continue
		}
		if filter.ClassificationStatus != "" && item.ClassificationStatus != filter.ClassificationStatus {
			continue
		}
		if filter.Subject != "" && !strings.EqualFold(item.Subject, filter.Subject) {
			continue
		}

		if filter.Chapter != "" && !strings.Contains(strings.ToLower(item.Chapter), strings.ToLower(filter.Chapter)) {
			continue
		}

		if filter.Keyword != "" {
			combined := searchtext.Normalize(item.QuestionCore + " " + item.SemanticSummary + " " + item.WrongSolution)
			for _, tag := range collectTypedTagRecords(item.Tags) {
				combined += "\n" + searchtext.Normalize(tag.Name)
			}
			if !strings.Contains(combined, searchtext.Normalize(filter.Keyword)) {
				continue
			}
		}

		if filter.MasteryStatus != "" && item.MasteryStatus != filter.MasteryStatus {
			continue
		}

		if filter.DifficultyLevel > 0 && item.DifficultyLevel != filter.DifficultyLevel {
			continue
		}

		if filter.SourceType != "" && item.SourceType != filter.SourceType {
			continue
		}

		if item.AnalysisStale && (len(filter.TagNames) > 0 || len(filter.TagIDs) > 0) {
			continue
		}
		if len(filter.TagNames) > 0 && !containsAnyTag(item.Tags, filter.TagNames) {
			continue
		}

		results = append(results, cloneQuestion(*item))
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].CreatedAt.Equal(results[j].CreatedAt) {
			return results[i].ID > results[j].ID
		}
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	total := len(results)
	start := (filter.Page - 1) * filter.PageSize
	if start >= total {
		return []model.WrongQuestion{}, total, nil
	}

	end := start + filter.PageSize
	if end > total {
		end = total
	}

	return results[start:end], total, nil
}

func (r *InMemoryQuestionRepository) DashboardMetrics(userID int64, now time.Time) (QuestionDashboardMetrics, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics := QuestionDashboardMetrics{
		MasteryStatusMap: make(map[string]int),
		SourceTypeMap:    make(map[string]int),
	}
	currentYear, currentMonth, currentDay := now.Date()

	for _, item := range r.items {
		if item.IsDeleted {
			continue
		}
		if userID > 0 && item.UserID != userID {
			continue
		}

		metrics.TotalQuestions++
		if item.MasteryStatus == "unmastered" {
			metrics.UnmasteredCount++
		}
		if item.SourceImageID != nil || strings.TrimSpace(item.SourceImageURL) != "" {
			metrics.ImageBoundCount++
		}
		metrics.MasteryStatusMap[item.MasteryStatus]++
		metrics.SourceTypeMap[item.SourceType]++

		year, month, day := item.CreatedAt.In(now.Location()).Date()
		if year == currentYear && month == currentMonth && day == currentDay {
			metrics.TodayAdded++
		}
	}

	return metrics, nil
}

func (r *InMemoryQuestionRepository) SoftDelete(id int64, deletedAt time.Time) (model.WrongQuestion, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.items[id]
	if !ok {
		return model.WrongQuestion{}, false, nil
	}

	item.IsDeleted = true
	item.DeletedAt = &deletedAt
	item.UpdatedAt = deletedAt

	return cloneQuestion(*item), true, nil
}

func containsAnyTag(tags model.TagCollections, filter []string) bool {
	all := append([]string{}, tags.KnowledgePoints...)
	all = append(all, tags.ProblemType...)
	all = append(all, tags.Method...)
	all = append(all, tags.MistakeReason...)

	index := make(map[string]struct{}, len(all))
	for _, item := range all {
		index[strings.ToLower(item)] = struct{}{}
	}

	for _, target := range filter {
		if _, ok := index[strings.ToLower(target)]; ok {
			return true
		}
	}

	return false
}

func cloneQuestion(question model.WrongQuestion) model.WrongQuestion {
	copyValue := question
	copyValue.Tags = model.TagCollections{
		KnowledgePoints: append([]string{}, question.Tags.KnowledgePoints...),
		ProblemType:     append([]string{}, question.Tags.ProblemType...),
		Method:          append([]string{}, question.Tags.Method...),
		MistakeReason:   append([]string{}, question.Tags.MistakeReason...),
	}

	if question.SourceImageID != nil {
		id := *question.SourceImageID
		copyValue.SourceImageID = &id
	}

	if question.DeletedAt != nil {
		deletedAt := *question.DeletedAt
		copyValue.DeletedAt = &deletedAt
	}

	return copyValue
}
