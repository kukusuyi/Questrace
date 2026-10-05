package service

import (
	"context"
	"net/http"

	"github.com/kukusuyi/Questrace/backend/internal/domain/dto"
	apperrors "github.com/kukusuyi/Questrace/backend/internal/pkg/errors"
	"github.com/kukusuyi/Questrace/backend/internal/repository"
)

type UserService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) GetMe(ctx context.Context) (dto.UserMeResponse, error) {
	userID, ok := GetUserID(ctx)
	if !ok {
		return dto.UserMeResponse{}, apperrors.New(http.StatusUnauthorized, 40100, "未获取到用户信息")
	}

	user, found, err := s.userRepo.GetByID(userID)
	if err != nil {
		return dto.UserMeResponse{}, err
	}
	if !found {
		return dto.UserMeResponse{}, apperrors.New(http.StatusNotFound, 40401, "用户不存在")
	}

	return dto.UserMeResponse{
		UserID:         user.ID,
		EducationStage: user.EducationStage,
		Username:       user.Username,
		Email:          user.Email,
		CreatedAt:      user.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

func ValidateEducationStage(stage string) error {
	if stage != "university" && stage != "highschool" {
		return apperrors.New(400, 40001, "请选择当前学习阶段：大学或高中")
	}
	return nil
}
func (s *UserService) UpdateEducationStage(ctx context.Context, stage string) (dto.UserMeResponse, error) {
	if err := ValidateEducationStage(stage); err != nil {
		return dto.UserMeResponse{}, err
	}
	me, err := s.GetMe(ctx)
	if err != nil {
		return me, err
	}
	if err = s.userRepo.UpdateEducationStage(me.UserID, stage); err != nil {
		return dto.UserMeResponse{}, err
	}
	return s.GetMe(ctx)
}
