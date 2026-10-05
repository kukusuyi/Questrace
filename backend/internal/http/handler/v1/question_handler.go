package v1

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/kukusuyi/Questrace/backend/internal/domain/dto"
	apperrors "github.com/kukusuyi/Questrace/backend/internal/pkg/errors"
	"github.com/kukusuyi/Questrace/backend/internal/pkg/pagination"
	"github.com/kukusuyi/Questrace/backend/internal/service"
)

type QuestionHandler struct {
	service *service.QuestionService
}

func NewQuestionHandler(service *service.QuestionService) *QuestionHandler {
	return &QuestionHandler{service: service}
}

func (h *QuestionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateWrongQuestionRequest
	if err := dto.DecodeJSON(r, &req); err != nil {
		dto.HandleError(w, err)
		return
	}

	response, err := h.service.Create(r.Context(), req)
	if err != nil {
		dto.HandleError(w, err)
		return
	}

	dto.WriteSuccess(w, response)
}

func (h *QuestionHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	page, pageSize = pagination.Normalize(page, pageSize)

	difficulty, _ := strconv.Atoi(r.URL.Query().Get("difficulty_level"))
	filter := dto.ListQuestionFilter{
		SubjectID: r.URL.Query().Get("subject_id"), CourseID: r.URL.Query().Get("course_id"), ClassificationStatus: r.URL.Query().Get("classification_status"),
		Page:            page,
		PageSize:        pageSize,
		Subject:         strings.TrimSpace(r.URL.Query().Get("subject")),
		Chapter:         strings.TrimSpace(r.URL.Query().Get("chapter")),
		Keyword:         strings.TrimSpace(r.URL.Query().Get("keyword")),
		TagIDs:          parseIDs(r.URL.Query().Get("tag_ids")),
		MasteryStatus:   strings.TrimSpace(r.URL.Query().Get("mastery_status")),
		DifficultyLevel: difficulty,
		SourceType:      strings.TrimSpace(r.URL.Query().Get("source_type")),
	}

	response, err := h.service.List(r.Context(), filter)
	if err != nil {
		dto.HandleError(w, err)
		return
	}

	dto.WriteSuccess(w, response)
}

func (h *QuestionHandler) Detail(w http.ResponseWriter, r *http.Request) {
	questionID, err := parsePathID(r.PathValue("questionID"))
	if err != nil {
		dto.HandleError(w, err)
		return
	}

	response, err := h.service.GetDetail(r.Context(), questionID)
	if err != nil {
		dto.HandleError(w, err)
		return
	}

	dto.WriteSuccess(w, response)
}

func (h *QuestionHandler) ExportPrint(w http.ResponseWriter, r *http.Request) {
	questionIDs := parseIDs(r.URL.Query().Get("question_ids"))
	exportMode := normalizeExportMode(r.URL.Query().Get("export_mode"))
	if exportMode == "" {
		dto.HandleError(w, apperrors.New(http.StatusBadRequest, 40001, "export_mode 不合法"))
		return
	}

	items, err := h.service.Export(r.Context(), questionIDs)
	if err != nil {
		dto.HandleError(w, err)
		return
	}

	if err := renderQuestionExportHTML(w, items, exportMode); err != nil {
		dto.HandleError(w, err)
	}
}

func (h *QuestionHandler) Update(w http.ResponseWriter, r *http.Request) {
	questionID, err := parsePathID(r.PathValue("questionID"))
	if err != nil {
		dto.HandleError(w, err)
		return
	}

	var req dto.UpdateWrongQuestionRequest
	if err := dto.DecodeJSON(r, &req); err != nil {
		dto.HandleError(w, err)
		return
	}

	response, err := h.service.Update(r.Context(), questionID, req)
	if err != nil {
		dto.HandleError(w, err)
		return
	}

	dto.WriteSuccess(w, response)
}

func (h *QuestionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	questionID, err := parsePathID(r.PathValue("questionID"))
	if err != nil {
		dto.HandleError(w, err)
		return
	}

	response, err := h.service.Delete(r.Context(), questionID)
	if err != nil {
		dto.HandleError(w, err)
		return
	}

	dto.WriteSuccess(w, response)
}

func (h *QuestionHandler) Similar(w http.ResponseWriter, r *http.Request) {
	questionID, err := parsePathID(r.PathValue("questionID"))
	if err != nil {
		dto.HandleError(w, err)
		return
	}

	var req dto.SimilarQuestionRequest
	if err := dto.DecodeJSON(r, &req); err != nil {
		dto.HandleError(w, err)
		return
	}

	response, err := h.service.Similar(r.Context(), questionID, req)
	if err != nil {
		dto.HandleError(w, err)
		return
	}

	dto.WriteSuccess(w, response)
}

func (h *QuestionHandler) SimilarByJSON(w http.ResponseWriter, r *http.Request) {
	var req dto.SimilarByJSONRequest
	if err := dto.DecodeJSON(r, &req); err != nil {
		dto.HandleError(w, err)
		return
	}

	response, err := h.service.SimilarByJSON(r.Context(), req)
	if err != nil {
		dto.HandleError(w, err)
		return
	}

	dto.WriteSuccess(w, response)
}

func parsePathID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, apperrors.New(http.StatusBadRequest, 40001, "路径参数不合法")
	}

	return id, nil
}

func parseIDs(raw string) []int64 {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	result := make([]int64, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil {
			continue
		}
		result = append(result, value)
	}

	return result
}

func normalizeExportMode(raw string) string {
	switch strings.TrimSpace(raw) {
	case "", exportModeWithAnswers:
		return exportModeWithAnswers
	case exportModeQuestionsOnly:
		return exportModeQuestionsOnly
	default:
		return ""
	}
}
