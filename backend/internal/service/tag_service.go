package service

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/kukusuyi/Questrace/backend/internal/domain/dto"
	"github.com/kukusuyi/Questrace/backend/internal/domain/enum"
	"github.com/kukusuyi/Questrace/backend/internal/domain/model"
	apperrors "github.com/kukusuyi/Questrace/backend/internal/pkg/errors"
	"github.com/kukusuyi/Questrace/backend/internal/pkg/validator"
	"github.com/kukusuyi/Questrace/backend/internal/repository"
)

type TagService struct {
	repo repository.TagRepository
}

func NewTagService(repo repository.TagRepository) *TagService {
	return &TagService{repo: repo}
}

func (s *TagService) List(ctx context.Context, tagType, keyword string, scope ...string) (dto.TagListResponse, error) {
	if err := validator.AllowEnum(tagType, "tag_type", enum.IsValidTagType); err != nil {
		return dto.TagListResponse{}, err
	}
	userID, err := RequireUserID(ctx)
	if err != nil {
		return dto.TagListResponse{}, err
	}

	subject, course := "", ""
	if len(scope) > 0 {
		subject = scope[0]
	}
	if len(scope) > 1 {
		course = scope[1]
	}
	items, err := s.repo.List(repository.TagFilter{
		SubjectID: subject, CourseID: course,
		UserID:  userID,
		TagType: tagType,
		Keyword: keyword,
	})
	if err != nil {
		return dto.TagListResponse{}, err
	}

	response := dto.TagListResponse{
		List: make([]dto.TagItem, 0, len(items)),
	}
	for _, item := range items {
		response.List = append(response.List, dto.TagItem{
			TagID:      item.ID,
			TagName:    item.TagName,
			TagType:    item.TagType,
			UsageCount: item.UsageCount,
			IsActive:   item.IsActive,
		})
	}

	return response, nil
}

func (s *TagService) Create(ctx context.Context, req dto.CreateTagRequest) (dto.TagItem, error) {
	if err := validator.RequireString(req.TagName, "tag_name"); err != nil {
		return dto.TagItem{}, err
	}
	if err := validator.AllowEnum(req.TagType, "tag_type", enum.IsValidTagType); err != nil {
		return dto.TagItem{}, err
	}
	userID, err := RequireUserID(ctx)
	if err != nil {
		return dto.TagItem{}, err
	}

	item, err := s.repo.Ensure(userID, req.TagType, req.TagName)
	if err != nil {
		return dto.TagItem{}, err
	}

	return dto.TagItem{
		TagID:      item.ID,
		TagName:    item.TagName,
		TagType:    item.TagType,
		UsageCount: item.UsageCount,
		IsActive:   item.IsActive,
	}, nil
}

func (s *TagService) Delete(ctx context.Context, id int64) error {
	userID, err := RequireUserID(ctx)
	if err != nil {
		return err
	}

	_, ok, err := s.repo.Delete(id, userID)
	if err != nil {
		return err
	}
	if !ok {
		return apperrors.New(http.StatusNotFound, 40404, "标签不存在")
	}

	return nil
}

func (s *TagService) Attach(userID int64, groups dto.TagGroups) (model.TagCollections, error) {
	normalized := normalizeTagGroups(groups)
	if err := s.ensureAll(userID, normalized); err != nil {
		return model.TagCollections{}, err
	}
	if err := s.adjustUsage(userID, normalized, 1); err != nil {
		return model.TagCollections{}, err
	}

	return normalized, nil
}

func (s *TagService) Replace(userID int64, old model.TagCollections, next dto.TagGroups) (model.TagCollections, error) {
	normalized := normalizeTagGroups(next)
	if err := s.ensureAll(userID, normalized); err != nil {
		return model.TagCollections{}, err
	}

	for _, change := range diffCollections(old, normalized) {
		if err := s.repo.AdjustUsage(userID, change.TagType, change.Name, change.Delta); err != nil {
			return model.TagCollections{}, err
		}
	}

	return normalized, nil
}

func (s *TagService) Detach(userID int64, groups model.TagCollections) error {
	return s.adjustUsage(userID, groups, -1)
}

func (s *TagService) ResolveTagNames(userID int64, ids []int64) ([]string, error) {
	return s.repo.ResolveNamesByIDs(userID, ids)
}

type tagUsageChange struct {
	TagType string
	Name    string
	Delta   int
}

func (s *TagService) ensureAll(userID int64, groups model.TagCollections) error {
	for _, pair := range collectTypedTags(groups) {
		if _, err := s.repo.Ensure(userID, pair.TagType, pair.Name); err != nil {
			return err
		}
	}

	return nil
}

func (s *TagService) adjustUsage(userID int64, groups model.TagCollections, delta int) error {
	for _, pair := range collectTypedTags(groups) {
		if err := s.repo.AdjustUsage(userID, pair.TagType, pair.Name, delta); err != nil {
			return err
		}
	}

	return nil
}

type typedTag struct {
	TagType string
	Name    string
}

func collectTypedTags(groups model.TagCollections) []typedTag {
	items := make([]typedTag, 0)
	for _, name := range groups.KnowledgePoints {
		items = append(items, typedTag{TagType: string(enum.TagTypeKnowledgePoint), Name: name})
	}
	for _, name := range groups.ProblemType {
		items = append(items, typedTag{TagType: string(enum.TagTypeProblemType), Name: name})
	}
	for _, name := range groups.Method {
		items = append(items, typedTag{TagType: string(enum.TagTypeMethod), Name: name})
	}
	for _, name := range groups.MistakeReason {
		items = append(items, typedTag{TagType: string(enum.TagTypeMistakeReason), Name: name})
	}

	return items
}

func normalizeTagGroups(groups dto.TagGroups) model.TagCollections {
	return model.TagCollections{
		KnowledgePoints: normalizeStringSlice(groups.KnowledgePoints),
		ProblemType:     normalizeStringSlice(groups.ProblemType),
		Method:          normalizeStringSlice(groups.Method),
		MistakeReason:   normalizeStringSlice(groups.MistakeReason),
	}
}

func normalizeStringSlice(input []string) []string {
	seen := make([]string, 0, len(input))
	for _, value := range input {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || slices.Contains(seen, trimmed) {
			continue
		}
		seen = append(seen, trimmed)
	}

	return seen
}

func diffCollections(old, next model.TagCollections) []tagUsageChange {
	oldSet := make(map[string]typedTag)
	nextSet := make(map[string]typedTag)

	for _, item := range collectTypedTags(old) {
		oldSet[item.TagType+"|"+item.Name] = item
	}
	for _, item := range collectTypedTags(next) {
		nextSet[item.TagType+"|"+item.Name] = item
	}

	changes := make([]tagUsageChange, 0)
	for key, item := range oldSet {
		if _, ok := nextSet[key]; !ok {
			changes = append(changes, tagUsageChange{TagType: item.TagType, Name: item.Name, Delta: -1})
		}
	}
	for key, item := range nextSet {
		if _, ok := oldSet[key]; !ok {
			changes = append(changes, tagUsageChange{TagType: item.TagType, Name: item.Name, Delta: 1})
		}
	}

	return changes
}
