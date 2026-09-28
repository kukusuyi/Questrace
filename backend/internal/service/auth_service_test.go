package service

import (
	"net/http"
	"testing"
	"time"

	"github.com/kukusuyi/Questrace/backend/internal/config"
	"github.com/kukusuyi/Questrace/backend/internal/domain/dto"
	"github.com/kukusuyi/Questrace/backend/internal/domain/model"
	apperrors "github.com/kukusuyi/Questrace/backend/internal/pkg/errors"
)

type authTestUserRepo struct{}

func (r *authTestUserRepo) UpdateEducationStage(id int64, stage string) error { return nil }

func (r *authTestUserRepo) Create(user model.User) (model.User, error) {
	user.ID = 2
	user.CreatedAt = time.Now()
	user.UpdatedAt = user.CreatedAt
	return user, nil
}

func (r *authTestUserRepo) GetByUsername(username string) (model.User, bool, error) {
	return model.User{}, false, nil
}

func (r *authTestUserRepo) GetByEmail(email string) (model.User, bool, error) {
	return model.User{}, false, nil
}

func (r *authTestUserRepo) GetByID(id int64) (model.User, bool, error) {
	return model.User{}, false, nil
}

func TestAuthServiceRegisterDisabled(t *testing.T) {
	service := NewAuthService(
		&authTestUserRepo{},
		config.JWTConfig{
			Secret:          "test-secret",
			ExpirationHours: 24,
		},
		config.AuthConfig{
			EnableRegistration: false,
		},
	)

	_, err := service.Register(dto.RegisterRequest{
		Username: "new_user",
		Password: "123456",
		Email:    "new_user@example.com",
	})
	if err == nil {
		t.Fatal("expected register to be blocked when registration is disabled")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Status != http.StatusForbidden {
		t.Fatalf("status=%d want=%d", appErr.Status, http.StatusForbidden)
	}
	if appErr.Code != 40301 {
		t.Fatalf("code=%d want=40301", appErr.Code)
	}
}

func TestRegisterRequiresExplicitEducationStage(t *testing.T) {
	s := NewAuthService(&authTestUserRepo{}, config.JWTConfig{Secret: "test-secret", ExpirationHours: 24}, config.AuthConfig{EnableRegistration: true})
	for _, stage := range []string{"", "unknown"} {
		if _, err := s.Register(dto.RegisterRequest{Username: "test", Email: "test@example.invalid", Password: "password123", EducationStage: stage}); err == nil {
			t.Fatal("missing or invalid stage accepted", stage)
		}
	}
	for _, stage := range []string{"university", "highschool"} {
		if _, err := s.Register(dto.RegisterRequest{Username: "test", Email: "test@example.invalid", Password: "password123", EducationStage: stage}); err != nil {
			t.Fatal(stage, err)
		}
	}
}
