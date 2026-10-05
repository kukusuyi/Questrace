package repository

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kukusuyi/Questrace/backend/internal/domain/model"
)

type TagFilter struct {
	SubjectID string
	CourseID  string
	UserID    int64
	TagType   string
	Keyword   string
}

type TagRepository interface {
	List(filter TagFilter) ([]model.Tag, error)
	Ensure(userID int64, tagType, tagName string) (model.Tag, error)
	AdjustUsage(userID int64, tagType, tagName string, delta int) error
	Delete(id, userID int64) (model.Tag, bool, error)
	ResolveNamesByIDs(userID int64, ids []int64) ([]string, error)
}

type InMemoryTagRepository struct {
	mu     sync.RWMutex
	nextID int64
	items  map[int64]*model.Tag
	index  map[string]int64
}

func NewInMemoryTagRepository() *InMemoryTagRepository {
	return &InMemoryTagRepository{
		nextID: 1,
		items:  make(map[int64]*model.Tag),
		index:  make(map[string]int64),
	}
}

func (r *InMemoryTagRepository) List(filter TagFilter) ([]model.Tag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]model.Tag, 0)
	for _, item := range r.items {
		if !item.IsActive {
			continue
		}
		if filter.UserID > 0 && item.UserID != filter.UserID {
			continue
		}

		if filter.TagType != "" && item.TagType != filter.TagType {
			continue
		}

		if filter.Keyword != "" && !strings.Contains(strings.ToLower(item.TagName), strings.ToLower(filter.Keyword)) {
			continue
		}

		result = append(result, cloneTag(*item))
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].UsageCount == result[j].UsageCount {
			return result[i].ID > result[j].ID
		}
		return result[i].UsageCount > result[j].UsageCount
	})

	return result, nil
}

func (r *InMemoryTagRepository) Ensure(userID int64, tagType, tagName string) (model.Tag, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := tagKey(userID, tagType, tagName)
	if id, ok := r.index[key]; ok {
		item := r.items[id]
		if !item.IsActive {
			item.IsActive = true
			item.DeletedAt = nil
			item.UpdatedAt = time.Now()
		}

		return cloneTag(*item), nil
	}

	now := time.Now()
	item := model.Tag{
		ID:         r.nextID,
		UserID:     userID,
		TagName:    strings.TrimSpace(tagName),
		TagType:    tagType,
		UsageCount: 0,
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	r.nextID++
	copyValue := cloneTag(item)
	r.items[item.ID] = &copyValue
	r.index[key] = item.ID

	return cloneTag(copyValue), nil
}

func (r *InMemoryTagRepository) AdjustUsage(userID int64, tagType, tagName string, delta int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := tagKey(userID, tagType, tagName)
	id, ok := r.index[key]
	if !ok {
		return nil
	}

	item := r.items[id]
	item.UsageCount += delta
	if item.UsageCount < 0 {
		item.UsageCount = 0
	}
	item.UpdatedAt = time.Now()

	return nil
}

func (r *InMemoryTagRepository) Delete(id, userID int64) (model.Tag, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.items[id]
	if !ok || (userID > 0 && item.UserID != userID) {
		return model.Tag{}, false, nil
	}

	now := time.Now()
	item.IsActive = false
	item.DeletedAt = &now
	item.UpdatedAt = now

	return cloneTag(*item), true, nil
}

func (r *InMemoryTagRepository) ResolveNamesByIDs(userID int64, ids []int64) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(ids))
	for _, id := range ids {
		item, ok := r.items[id]
		if !ok || !item.IsActive || (userID > 0 && item.UserID != userID) {
			continue
		}
		names = append(names, item.TagName)
	}

	return names, nil
}

func tagKey(userID int64, tagType, tagName string) string {
	return strings.ToLower(strings.TrimSpace(tagType) + "|" + strings.TrimSpace(tagName) + "|" + strconv.FormatInt(userID, 10))
}

func cloneTag(tag model.Tag) model.Tag {
	copyValue := tag
	if tag.DeletedAt != nil {
		value := *tag.DeletedAt
		copyValue.DeletedAt = &value
	}

	return copyValue
}
