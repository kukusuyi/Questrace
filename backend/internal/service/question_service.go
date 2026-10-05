package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/kukusuyi/Questrace/backend/internal/domain/dto"
	"github.com/kukusuyi/Questrace/backend/internal/domain/enum"
	"github.com/kukusuyi/Questrace/backend/internal/domain/model"
	apperrors "github.com/kukusuyi/Questrace/backend/internal/pkg/errors"
	"github.com/kukusuyi/Questrace/backend/internal/pkg/pagination"
	"github.com/kukusuyi/Questrace/backend/internal/pkg/timeutil"
	"github.com/kukusuyi/Questrace/backend/internal/pkg/validator"
	"github.com/kukusuyi/Questrace/backend/internal/repository"
)

type QuestionService struct {
	Subjects      *SubjectService
	repo          repository.QuestionRepository
	fileService   *FileService
	tagService    *TagService
	vectorService *VectorService
}

func NewQuestionService(
	repo repository.QuestionRepository,
	fileService *FileService,
	tagService *TagService,
	vectorService *VectorService,
) *QuestionService {
	return &QuestionService{
		repo:          repo,
		fileService:   fileService,
		tagService:    tagService,
		vectorService: vectorService,
	}
}

func (s *QuestionService) Create(ctx context.Context, req dto.CreateWrongQuestionRequest) (dto.CreateWrongQuestionResponse, error) {
	if req.SourceImageID != nil {
		u, err := s.fileService.OwnedImageURL(ctx, *req.SourceImageID)
		if err != nil {
			return dto.CreateWrongQuestionResponse{}, err
		}
		req.SourceImageURL = u
	} else {
		req.SourceImageURL = ""
	}
	if err := validateCreateQuestion(req); err != nil {
		return dto.CreateWrongQuestionResponse{}, err
	}
	userID, err := RequireUserID(ctx)
	if err != nil {
		return dto.CreateWrongQuestionResponse{}, err
	}

	now := time.Now()
	question := model.WrongQuestion{
		UserID:           userID,
		Subject:          strings.TrimSpace(req.Subject),
		Chapter:          strings.TrimSpace(req.Chapter),
		QuestionCore:     strings.TrimSpace(req.QuestionJSON.QuestionCore),
		StandardSolution: strings.TrimSpace(req.QuestionJSON.StandardSolution),
		WrongSolution:    strings.TrimSpace(req.QuestionJSON.WrongSolution),
		SemanticSummary:  strings.TrimSpace(req.SemanticSummary),
		MistakeSummary:   strings.TrimSpace(req.MistakeSummary),
		DifficultyLevel:  req.DifficultyLevel,
		MasteryStatus:    defaultMasteryStatus(req.MasteryStatus),
		SourceType:       req.SourceType,
		SourceImageID:    req.SourceImageID,
		SourceImageURL:   strings.TrimSpace(req.SourceImageURL),

		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.Subjects.Apply(ctx, &question, req.SubjectID, req.CourseID, req.Chapter); err != nil {
		return dto.CreateWrongQuestionResponse{}, err
	}
	tags, err := s.tagService.Attach(userID, req.Tags)
	if err != nil {
		return dto.CreateWrongQuestionResponse{}, err
	}

	question.Tags = tags
	question.AnalysisStale = req.AnalysisStale && !req.AnalysisConfirmed
	created, err := s.repo.Create(question)
	if err != nil {
		_ = s.tagService.Detach(userID, tags)
		return dto.CreateWrongQuestionResponse{}, err
	}

	if err := s.vectorService.Upsert(created); err != nil {
		return dto.CreateWrongQuestionResponse{}, err
	}

	return dto.CreateWrongQuestionResponse{QuestionID: created.ID}, nil
}

func (s *QuestionService) List(ctx context.Context, filter dto.ListQuestionFilter) (dto.PageResult[dto.QuestionListItem], error) {
	page, pageSize := pagination.Normalize(filter.Page, filter.PageSize)
	filter.Page = page
	filter.PageSize = pageSize
	userID, err := RequireUserID(ctx)
	if err != nil {
		return dto.PageResult[dto.QuestionListItem]{}, err
	}

	if err := validator.AllowEnum(filter.MasteryStatus, "mastery_status", enum.IsValidMasteryStatus); err != nil {
		return dto.PageResult[dto.QuestionListItem]{}, err
	}
	if err := validator.AllowEnum(filter.SourceType, "source_type", enum.IsValidSourceType); err != nil {
		return dto.PageResult[dto.QuestionListItem]{}, err
	}
	if err := validator.Difficulty(filter.DifficultyLevel); err != nil {
		return dto.PageResult[dto.QuestionListItem]{}, err
	}

	tagNames, err := s.tagService.ResolveTagNames(userID, filter.TagIDs)
	if err != nil {
		return dto.PageResult[dto.QuestionListItem]{}, err
	}

	items, total, err := s.repo.List(repository.QuestionFilter{
		UserID:    userID,
		Page:      filter.Page,
		PageSize:  filter.PageSize,
		SubjectID: filter.SubjectID, CourseID: filter.CourseID, ClassificationStatus: filter.ClassificationStatus,
		Subject:         filter.Subject,
		Chapter:         filter.Chapter,
		Keyword:         filter.Keyword,
		TagNames:        tagNames,
		TagIDs:          filter.TagIDs,
		MasteryStatus:   filter.MasteryStatus,
		DifficultyLevel: filter.DifficultyLevel,
		SourceType:      filter.SourceType,
	})
	if err != nil {
		return dto.PageResult[dto.QuestionListItem]{}, err
	}

	list := make([]dto.QuestionListItem, 0, len(items))
	for _, item := range items {
		list = append(list, dto.QuestionListItem{
			QuestionID:     item.ID,
			QuestionCore:   item.QuestionCore,
			SourceImageID:  item.SourceImageID,
			SourceImageURL: item.SourceImageURL,
			Subject:        item.Subject,
			SubjectID:      item.SubjectID, CourseID: item.CourseID, ClassificationStatus: item.ClassificationStatus, AnalysisStale: item.AnalysisStale, Revision: item.Revision,
			Chapter:         item.Chapter,
			Tags:            toTagGroups(item.Tags),
			DifficultyLevel: item.DifficultyLevel,
			MasteryStatus:   item.MasteryStatus,
			CreatedAt:       timeutil.Format(item.CreatedAt),
		})
	}

	return dto.PageResult[dto.QuestionListItem]{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *QuestionService) GetDetail(ctx context.Context, id int64) (dto.QuestionDetail, error) {
	question, err := s.getActiveQuestion(ctx, id)
	if err != nil {
		return dto.QuestionDetail{}, err
	}

	return dto.QuestionDetail{
		QuestionID:       question.ID,
		QuestionCore:     question.QuestionCore,
		StandardSolution: question.StandardSolution,
		WrongSolution:    question.WrongSolution,
		SemanticSummary:  question.SemanticSummary,
		MistakeSummary:   question.MistakeSummary,
		SourceType:       question.SourceType,
		SourceImageID:    question.SourceImageID,
		SourceImageURL:   question.SourceImageURL,
		Subject:          question.Subject,
		SubjectID:        question.SubjectID, CourseID: question.CourseID, ClassificationStatus: question.ClassificationStatus, AnalysisStale: question.AnalysisStale, Revision: question.Revision,
		Chapter:         question.Chapter,
		Tags:            toTagGroups(question.Tags),
		DifficultyLevel: question.DifficultyLevel,
		MasteryStatus:   question.MasteryStatus,
		CreatedAt:       timeutil.Format(question.CreatedAt),
		UpdatedAt:       timeutil.Format(question.UpdatedAt),
	}, nil
}

func (s *QuestionService) Export(ctx context.Context, ids []int64) ([]dto.QuestionExportItem, error) {
	if len(ids) == 0 {
		return nil, apperrors.New(http.StatusBadRequest, 40001, "至少选择一道错题")
	}
	if len(ids) > 50 {
		return nil, apperrors.New(http.StatusBadRequest, 40001, "单次最多导出 50 道错题")
	}

	normalized := uniquePositiveIDs(ids)
	if len(normalized) == 0 {
		return nil, apperrors.New(http.StatusBadRequest, 40001, "至少选择一道合法错题")
	}

	items := make([]dto.QuestionExportItem, 0, len(normalized))
	for _, id := range normalized {
		question, err := s.getActiveQuestion(ctx, id)
		if err != nil {
			return nil, err
		}

		items = append(items, dto.QuestionExportItem{
			QuestionID:       question.ID,
			QuestionCore:     question.QuestionCore,
			StandardSolution: question.StandardSolution,
			WrongSolution:    question.WrongSolution,
			SemanticSummary:  question.SemanticSummary,
			MistakeSummary:   question.MistakeSummary,
			SourceType:       question.SourceType,
			SourceImageURL:   question.SourceImageURL,
			Subject:          question.Subject,
			SubjectID:        question.SubjectID, CourseID: question.CourseID, ClassificationStatus: question.ClassificationStatus, AnalysisStale: question.AnalysisStale, Revision: question.Revision,
			Chapter:         question.Chapter,
			Tags:            toTagGroups(question.Tags),
			DifficultyLevel: question.DifficultyLevel,
			MasteryStatus:   question.MasteryStatus,
			CreatedAt:       timeutil.Format(question.CreatedAt),
			UpdatedAt:       timeutil.Format(question.UpdatedAt),
		})
	}

	return items, nil
}

func (s *QuestionService) Update(ctx context.Context, id int64, req dto.UpdateWrongQuestionRequest) (dto.UpdateWrongQuestionResponse, error) {
	if req.SourceImageID != nil {
		u, err := s.fileService.OwnedImageURL(ctx, *req.SourceImageID)
		if err != nil {
			return dto.UpdateWrongQuestionResponse{}, err
		}
		req.SourceImageURL = u
	} else {
		req.SourceImageURL = ""
	}
	if err := validateUpdateQuestion(req); err != nil {
		return dto.UpdateWrongQuestionResponse{}, err
	}
	userID, err := RequireUserID(ctx)
	if err != nil {
		return dto.UpdateWrongQuestionResponse{}, err
	}

	current, err := s.getActiveQuestion(ctx, id)
	if err != nil {
		return dto.UpdateWrongQuestionResponse{}, err
	}

	if req.Revision > 0 && req.Revision != current.Revision {
		return dto.UpdateWrongQuestionResponse{}, apperrors.New(409, 40901, "题目已被修改，请刷新后重试")
	}
	oldSubjectID, oldCourseID, oldStatus, oldChapter := current.SubjectID, current.CourseID, current.ClassificationStatus, current.Chapter
	if err := s.Subjects.Apply(ctx, &current, req.SubjectID, req.CourseID, req.Chapter); err != nil {
		return dto.UpdateWrongQuestionResponse{}, err
	}
	if req.SubjectID == "" && oldSubjectID == "" && oldStatus == "legacy_pending" {
		current.ClassificationStatus = oldStatus
	}
	current.AnalysisStale = current.AnalysisStale || oldSubjectID != current.SubjectID || oldCourseID != current.CourseID || oldChapter != current.Chapter || current.QuestionCore != strings.TrimSpace(req.QuestionJSON.QuestionCore) || current.WrongSolution != strings.TrimSpace(req.QuestionJSON.WrongSolution)
	if req.AnalysisConfirmed {
		current.AnalysisStale = false
	}
	if req.SubjectID == "" {
		current.Subject = strings.TrimSpace(req.Subject)
	}
	current.Chapter = strings.TrimSpace(req.Chapter)
	current.QuestionCore = strings.TrimSpace(req.QuestionJSON.QuestionCore)
	current.StandardSolution = strings.TrimSpace(req.QuestionJSON.StandardSolution)
	current.WrongSolution = strings.TrimSpace(req.QuestionJSON.WrongSolution)
	current.SemanticSummary = strings.TrimSpace(req.SemanticSummary)
	current.MistakeSummary = strings.TrimSpace(req.MistakeSummary)
	current.DifficultyLevel = req.DifficultyLevel
	current.MasteryStatus = defaultMasteryStatus(req.MasteryStatus)
	current.SourceImageID = req.SourceImageID
	current.SourceImageURL = strings.TrimSpace(req.SourceImageURL)
	tags, err := s.tagService.Replace(userID, current.Tags, req.Tags)
	if err != nil {
		return dto.UpdateWrongQuestionResponse{}, err
	}

	current.Tags = tags
	current.UpdatedAt = time.Now()

	updated, err := s.repo.Update(current)
	if err != nil {
		return dto.UpdateWrongQuestionResponse{}, err
	}

	if err := s.vectorService.Upsert(updated); err != nil {
		return dto.UpdateWrongQuestionResponse{}, err
	}

	return dto.UpdateWrongQuestionResponse{
		QuestionID: updated.ID,
		Updated:    true,
	}, nil
}

func (s *QuestionService) Delete(ctx context.Context, id int64) (dto.DeleteWrongQuestionResponse, error) {
	question, err := s.getActiveQuestion(ctx, id)
	if err != nil {
		return dto.DeleteWrongQuestionResponse{}, err
	}

	if err := s.vectorService.Delete(question.ID); err != nil {
		return dto.DeleteWrongQuestionResponse{}, err
	}
	if err := s.tagService.Detach(question.UserID, question.Tags); err != nil {
		return dto.DeleteWrongQuestionResponse{}, err
	}

	_, ok, err := s.repo.SoftDelete(id, time.Now())
	if err != nil {
		return dto.DeleteWrongQuestionResponse{}, err
	}
	if !ok {
		return dto.DeleteWrongQuestionResponse{}, apperrors.New(http.StatusNotFound, 40404, "错题不存在")
	}

	return dto.DeleteWrongQuestionResponse{
		QuestionID: id,
		Deleted:    true,
	}, nil
}

func (s *QuestionService) Similar(ctx context.Context, id int64, req dto.SimilarQuestionRequest) (dto.SimilarQuestionResponse, error) {
	question, err := s.getActiveQuestion(ctx, id)
	if err != nil {
		return dto.SimilarQuestionResponse{}, err
	}

	question.RecallScope = req.RecallScope
	return s.similarByBase(ctx, question, req.VectorType, req.Limit, req.UseTagFilter)
}

func (s *QuestionService) SimilarByJSON(ctx context.Context, req dto.SimilarByJSONRequest) (dto.SimilarQuestionResponse, error) {
	if err := validator.RequireString(req.QuestionJSON.QuestionCore, "question_json.question_core"); err != nil {
		return dto.SimilarQuestionResponse{}, err
	}
	userID, err := RequireUserID(ctx)
	if err != nil {
		return dto.SimilarQuestionResponse{}, err
	}

	base := model.WrongQuestion{
		UserID:          userID,
		QuestionCore:    strings.TrimSpace(req.QuestionJSON.QuestionCore),
		WrongSolution:   strings.TrimSpace(req.QuestionJSON.WrongSolution),
		SemanticSummary: strings.TrimSpace(req.QuestionJSON.QuestionCore),
		MistakeSummary:  strings.TrimSpace(req.QuestionJSON.WrongSolution),
		Tags:            normalizeTagGroups(req.Tags),
	}

	if req.ClassificationStatus != "" && req.ClassificationStatus != "confirmed" {
		return dto.SimilarQuestionResponse{}, apperrors.New(409, 40902, "请先确认题目学科，再查找相似题")
	}
	if err := s.Subjects.Apply(ctx, &base, req.SubjectID, req.CourseID, req.Chapter); err != nil {
		return dto.SimilarQuestionResponse{}, err
	}
	base.AnalysisStale = req.AnalysisStale
	base.RecallScope = req.RecallScope
	return s.similarByBase(ctx, base, req.VectorType, req.Limit, req.UseTagFilter)
}

func (s *QuestionService) similarByBase(ctx context.Context, base model.WrongQuestion, vectorType string, limit int, useTagFilter bool) (dto.SimilarQuestionResponse, error) {
	userID, err := RequireUserID(ctx)
	if err != nil {
		return dto.SimilarQuestionResponse{}, err
	}
	if limit <= 0 {
		limit = 10
	}
	if err := validator.AllowEnum(vectorType, "vector_type", enum.IsValidVectorType); err != nil {
		return dto.SimilarQuestionResponse{}, err
	}
	if vectorType == "" {
		vectorType = string(enum.VectorTypeSemantic)
	}

	searchResults, err := s.vectorService.Search(base, vectorType, limit, useTagFilter)
	if err != nil {
		return dto.SimilarQuestionResponse{}, err
	}

	results := make([]dto.SimilarQuestionItem, 0, len(searchResults))
	for _, item := range searchResults {
		candidate, ok := s.repo.GetByID(item.QuestionID)
		if !ok || candidate.IsDeleted || candidate.UserID != userID || candidate.SubjectID != base.SubjectID || candidate.ClassificationStatus != "confirmed" || (base.SubjectID == "cs408" && base.RecallScope != "subject" && candidate.CourseID != base.CourseID) {
			continue
		}

		matchedTags := overlapTags(base.Tags, candidate.Tags)
		if base.AnalysisStale || candidate.AnalysisStale {
			matchedTags = []string{}
		}
		if useTagFilter && len(matchedTags) == 0 {
			continue
		}

		results = append(results, dto.SimilarQuestionItem{
			SubjectID: candidate.SubjectID, Subject: candidate.Subject, CourseID: candidate.CourseID, Chapter: candidate.Chapter, ClassificationStatus: candidate.ClassificationStatus,
			QuestionID:     candidate.ID,
			Score:          item.Score,
			SimilarityType: similarityType(useTagFilter, vectorType, matchedTags),
			QuestionCore:   candidate.QuestionCore,
			SourceImageID:  candidate.SourceImageID,
			SourceImageURL: candidate.SourceImageURL,
			MatchedTags:    matchedTags,
			Reason:         buildSimilarityReason(matchedTags, vectorType),
			Tags:           toTagGroups(candidate.Tags),
		})
	}

	status := "ready"
	if len(results) == 0 {
		status = "no_matches"
		if s.vectorService.Local.IndexPending(base) {
			status = "indexing"
		}
	}
	return dto.SimilarQuestionResponse{List: results, Status: status}, nil
}

func (s *QuestionService) getActiveQuestion(ctx context.Context, id int64) (model.WrongQuestion, error) {
	userID, err := RequireUserID(ctx)
	if err != nil {
		return model.WrongQuestion{}, err
	}

	question, ok := s.repo.GetByID(id)
	if !ok || question.IsDeleted || question.UserID != userID {
		return model.WrongQuestion{}, apperrors.New(http.StatusNotFound, 40404, "错题不存在")
	}

	return question, nil
}

func validateCreateQuestion(req dto.CreateWrongQuestionRequest) error {
	if err := validator.AllowEnum(req.SourceType, "source_type", enum.IsValidSourceType); err != nil {
		return err
	}

	if err := validator.RequireString(req.QuestionJSON.QuestionCore, "question_json.question_core"); err != nil {
		return err
	}

	if err := validator.AllowEnum(req.MasteryStatus, "mastery_status", enum.IsValidMasteryStatus); err != nil {
		return err
	}
	if err := validator.Difficulty(req.DifficultyLevel); err != nil {
		return err
	}
	if req.SourceType == string(enum.SourceTypeImage) {
		if req.SourceImageID == nil || strings.TrimSpace(req.SourceImageURL) == "" {
			return apperrors.New(http.StatusBadRequest, 40001, "source_type=image 时必须提供 source_image_id 和 source_image_url")
		}
	}

	return nil
}

func validateUpdateQuestion(req dto.UpdateWrongQuestionRequest) error {

	if err := validator.RequireString(req.QuestionJSON.QuestionCore, "question_json.question_core"); err != nil {
		return err
	}

	if err := validator.AllowEnum(req.MasteryStatus, "mastery_status", enum.IsValidMasteryStatus); err != nil {
		return err
	}
	if err := validator.Difficulty(req.DifficultyLevel); err != nil {
		return err
	}

	return nil
}

func defaultMasteryStatus(value string) string {
	if strings.TrimSpace(value) == "" {
		return string(enum.MasteryStatusUnmastered)
	}
	return value
}

func toTagGroups(tags model.TagCollections) dto.TagGroups {
	return dto.TagGroups{
		KnowledgePoints: append([]string{}, tags.KnowledgePoints...),
		ProblemType:     append([]string{}, tags.ProblemType...),
		Method:          append([]string{}, tags.Method...),
		MistakeReason:   append([]string{}, tags.MistakeReason...),
	}
}

func overlapTags(a, b model.TagCollections) []string {
	index := make(map[string]struct{})
	for _, group := range [][]string{a.KnowledgePoints, a.ProblemType, a.Method, a.MistakeReason} {
		for _, item := range group {
			index[item] = struct{}{}
		}
	}

	matched := make([]string, 0)
	for _, group := range [][]string{b.KnowledgePoints, b.ProblemType, b.Method, b.MistakeReason} {
		for _, item := range group {
			if _, ok := index[item]; ok && !containsString(matched, item) {
				matched = append(matched, item)
			}
		}
	}

	return matched
}

func similarityType(useTagFilter bool, vectorType string, matchedTags []string) string {
	if useTagFilter && len(matchedTags) > 0 {
		return "hybrid"
	}
	return vectorType
}

func buildSimilarityReason(matchedTags []string, vectorType string) string {
	if len(matchedTags) > 0 {
		return "命中相同标签：" + strings.Join(matchedTags, "、")
	}
	if vectorType == string(enum.VectorTypeMistake) {
		return "根据错因摘要进行了近似匹配"
	}
	return "根据题目语义摘要进行了近似匹配"
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}

func uniquePositiveIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}

	return result
}
