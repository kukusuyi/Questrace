package v1

import (
	"net/http"

	"github.com/kukusuyi/Questrace/backend/internal/domain/dto"
	"github.com/kukusuyi/Questrace/backend/internal/service"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.GetMe(r.Context())
	if err != nil {
		dto.HandleError(w, err)
		return
	}

	dto.WriteSuccess(w, resp)
}

func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		EducationStage string `json:"education_stage"`
	}
	if err := dto.DecodeJSON(r, &in); err != nil {
		dto.HandleError(w, err)
		return
	}
	out, err := h.service.UpdateEducationStage(r.Context(), in.EducationStage)
	if err != nil {
		dto.HandleError(w, err)
		return
	}
	dto.WriteSuccess(w, out)
}
