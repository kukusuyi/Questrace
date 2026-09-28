package service

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kukusuyi/Questrace/backend/internal/config"
	"github.com/kukusuyi/Questrace/backend/internal/domain/dto"
	"github.com/kukusuyi/Questrace/backend/internal/domain/model"
	apperrors "github.com/kukusuyi/Questrace/backend/internal/pkg/errors"
	"github.com/kukusuyi/Questrace/backend/internal/repository"
)

type AuthService struct {
	userRepo           repository.UserRepository
	jwtCfg             config.JWTConfig
	enableRegistration bool
}

func NewAuthService(userRepo repository.UserRepository, jwtCfg config.JWTConfig, authCfg config.AuthConfig) *AuthService {
	return &AuthService{
		userRepo:           userRepo,
		jwtCfg:             jwtCfg,
		enableRegistration: authCfg.EnableRegistration,
	}
}

func (s *AuthService) Register(req dto.RegisterRequest) (dto.RegisterResponse, error) {
	if !s.enableRegistration {
		return dto.RegisterResponse{}, apperrors.New(http.StatusForbidden, 40301, "当前服务未开放用户注册")
	}

	if err := ValidateEducationStage(req.EducationStage); err != nil {
		return dto.RegisterResponse{}, err
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)

	if req.Username == "" {
		return dto.RegisterResponse{}, apperrors.New(http.StatusBadRequest, 40001, "用户名不能为空")
	}
	if len(req.Password) < 6 {
		return dto.RegisterResponse{}, apperrors.New(http.StatusBadRequest, 40001, "密码长度不能少于6位")
	}
	if req.Email == "" {
		return dto.RegisterResponse{}, apperrors.New(http.StatusBadRequest, 40001, "邮箱不能为空")
	}

	if _, found, err := s.userRepo.GetByUsername(req.Username); err != nil {
		return dto.RegisterResponse{}, err
	} else if found {
		return dto.RegisterResponse{}, apperrors.New(http.StatusConflict, 40901, "用户名已存在")
	}

	if _, found, err := s.userRepo.GetByEmail(req.Email); err != nil {
		return dto.RegisterResponse{}, err
	} else if found {
		return dto.RegisterResponse{}, apperrors.New(http.StatusConflict, 40902, "邮箱已被注册")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return dto.RegisterResponse{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.userRepo.Create(model.User{
		Username:       req.Username,
		Email:          req.Email,
		PasswordHash:   string(hash),
		Role:           "user",
		EducationStage: req.EducationStage,
	})
	if err != nil {
		return dto.RegisterResponse{}, fmt.Errorf("create user: %w", err)
	}

	token, err := s.generateJWT(user.ID)
	if err != nil {
		return dto.RegisterResponse{}, err
	}

	return dto.RegisterResponse{
		UserID:   user.ID,
		Username: user.Username,
		Token:    token,
	}, nil
}

func (s *AuthService) Login(req dto.LoginRequest) (dto.LoginResponse, error) {
	req.Username = strings.TrimSpace(req.Username)

	if req.Username == "" {
		return dto.LoginResponse{}, apperrors.New(http.StatusBadRequest, 40001, "用户名不能为空")
	}
	if req.Password == "" {
		return dto.LoginResponse{}, apperrors.New(http.StatusBadRequest, 40001, "密码不能为空")
	}

	user, found, err := s.userRepo.GetByUsername(req.Username)
	if err != nil {
		return dto.LoginResponse{}, err
	}
	if !found {
		return dto.LoginResponse{}, apperrors.New(http.StatusUnauthorized, 40101, "用户名或密码错误")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return dto.LoginResponse{}, apperrors.New(http.StatusUnauthorized, 40101, "用户名或密码错误")
	}

	token, err := s.generateJWT(user.ID)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	return dto.LoginResponse{
		UserID:   user.ID,
		Username: user.Username,
		Token:    token,
	}, nil
}

func (s *AuthService) ValidateJWT(tokenString string) (int64, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtCfg.Secret), nil
	})
	if err != nil {
		return 0, apperrors.New(http.StatusUnauthorized, 40100, "无效的认证令牌")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, apperrors.New(http.StatusUnauthorized, 40100, "无效的认证令牌")
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, apperrors.New(http.StatusUnauthorized, 40100, "无效的认证令牌")
	}

	return int64(userIDFloat), nil
}

func (s *AuthService) generateJWT(userID int64) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Duration(s.jwtCfg.ExpirationHours) * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtCfg.Secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return tokenString, nil
}
