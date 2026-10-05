package v1

import (
	"net/http"

	"github.com/kukusuyi/Questrace/backend/internal/domain/dto"
	"github.com/kukusuyi/Questrace/backend/internal/service"
)

type AIHandler struct {
	service *service.AIService
}

func NewAIHandler(service *service.AIService) *AIHandler {
	return &AIHandler{service: service}
}

func (h *AIHandler) AnalyzeWrongQuestion(w http.ResponseWriter, r *http.Request) {
	var req dto.AnalyzeWrongQuestionRequest
	if err := dto.DecodeJSON(r, &req); err != nil {
		dto.HandleError(w, err)
		return
	}

	response, err := h.service.Analyze(r.Context(), req)
	if err != nil {
		dto.HandleError(w, err)
		return
	}

	dto.WriteSuccess(w, response)
}

func (h *AIHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	dto.WriteSuccess(w, h.service.ListProviders())
}

func (h *AIHandler) ListProviderModels(w http.ResponseWriter, r *http.Request) {
	providerName := r.PathValue("providerName")

	response, err := h.service.ListProviderModels(r.Context(), providerName)
	if err != nil {
		dto.HandleError(w, err)
		return
	}

	dto.WriteSuccess(w, response)
}

func (h *AIHandler) ListChapters(w http.ResponseWriter, r *http.Request) {
	id, course := r.URL.Query().Get("subject_id"), r.URL.Query().Get("course_id")
	response := dto.AIChapterListResponse{List: []string{}}
	if id != "" {
		x, err := h.service.Subjects.Resolve(r.Context(), id, course, "")
		if err != nil {
			dto.HandleError(w, err)
			return
		}
		response.List = x.Chapters
		for _, c := range x.Courses {
			if c.ID == course {
				response.List = c.Chapters
			}
		}
	}

	dto.WriteSuccess(w, response)
}
