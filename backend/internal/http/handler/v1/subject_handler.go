package v1

import (
	"github.com/kukusuyi/Questrace/backend/internal/domain/dto"
	"github.com/kukusuyi/Questrace/backend/internal/service"
	"net/http"
)

func (h *AIHandler) ListSubjects(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.Subjects.List(r.Context())
	if err != nil {
		dto.HandleError(w, err)
		return
	}
	stage, err := h.service.Subjects.EducationStage(r.Context())
	if err != nil {
		dto.HandleError(w, err)
		return
	}
	for i := range items {
		items[i].Recommended = items[i].EducationStage == stage
	}
	dto.WriteSuccess(w, map[string]any{"list": items, "education_stage": stage})
}
func (h *AIHandler) CreateSubject(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if err := dto.DecodeJSON(r, &in); err != nil {
		dto.HandleError(w, err)
		return
	}
	out, err := h.service.Subjects.Create(r.Context(), in.Name)
	if err != nil {
		dto.HandleError(w, err)
		return
	}
	dto.WriteSuccess(w, out)
}
func (h *QuestionHandler) Reclassify(w http.ResponseWriter, r *http.Request) {
	var in service.ClassificationBatch
	if err := dto.DecodeJSON(r, &in); err != nil {
		dto.HandleError(w, err)
		return
	}
	if err := h.service.Subjects.Reclassify(r.Context(), in); err != nil {
		dto.HandleError(w, err)
		return
	}
	dto.WriteSuccess(w, map[string]any{"updated": len(in.Items)})
}
