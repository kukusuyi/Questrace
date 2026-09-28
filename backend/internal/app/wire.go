package app

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/kukusuyi/Questrace/backend/internal/config"
	httpx "github.com/kukusuyi/Questrace/backend/internal/http"
	v1 "github.com/kukusuyi/Questrace/backend/internal/http/handler/v1"
	aiinfra "github.com/kukusuyi/Questrace/backend/internal/infra/ai"
	"github.com/kukusuyi/Questrace/backend/internal/infra/ocr"
	"github.com/kukusuyi/Questrace/backend/internal/infra/oss"
	"github.com/kukusuyi/Questrace/backend/internal/repository"
	"github.com/kukusuyi/Questrace/backend/internal/service"
)

func BuildHTTPHandler(cfg config.Config, appLogger *slog.Logger, db *sql.DB) (http.Handler, error) {
	questionRepo := repository.NewSQLiteQuestionRepository(db)
	tagRepo := repository.NewSQLiteTagRepository(db)
	fileRepo := repository.NewSQLiteFileRepository(db)

	userRepo := repository.NewSQLiteUserRepository(db)
	aiAnalysisRecordRepo := repository.NewSQLiteAIAnalysisRecordRepository(db)

	authService := service.NewAuthService(userRepo, cfg.JWT, cfg.Auth)
	userService := service.NewUserService(userRepo)
	tagService := service.NewTagService(tagRepo)
	dashboardService := service.NewDashboardService(questionRepo, tagRepo)
	objectStorage, err := oss.NewClient(cfg.File)
	if err != nil {
		return nil, err
	}
	if err := objectStorage.EnsureReady(context.Background()); err != nil {
		return nil, err
	}
	fileService := service.NewFileService(fileRepo, objectStorage, cfg.File, cfg.App.Env)
	var ocrClient ocr.OCRClient
	if cfg.ImageOcr.APIKey != "" {
		ocrClient = ocr.NewQwenOCRClientWithLogger(cfg.ImageOcr.APIKey, cfg.ImageOcr.Model, appLogger)
	}
	ocrService, err := service.NewOCRService(ocrClient, 0)
	if err != nil {
		return nil, err
	}
	llmRegistry, err := aiinfra.NewRegistry(cfg.Models)
	if err != nil {
		return nil, err
	}
	aiService, err := service.NewAIService(
		llmRegistry,
		aiAnalysisRecordRepo,
		cfg.ImageOcr.Name,
		cfg.ImageOcr.Model,
	)
	if err != nil {
		return nil, err
	}
	vectorService := &service.VectorService{Local: &service.LocalVector{DB: db, Config: func() config.EmbeddingModelConfig { return cfg.EmbeddingModel }}}
	questionService := service.NewQuestionService(questionRepo, fileService, tagService, vectorService)
	aiService.Subjects = &service.SubjectService{DB: db}
	questionService.Subjects = aiService.Subjects
	mobileService := service.NewMobileService(cfg.MobileVersion, cfg.File)

	handlers := httpx.V1Handlers{
		Auth:      v1.NewAuthHandler(authService),
		User:      v1.NewUserHandler(userService),
		Dashboard: v1.NewDashboardHandler(dashboardService),
		Tag:       v1.NewTagHandler(tagService),
		File:      v1.NewFileHandler(fileService),
		OCR:       v1.NewOCRHandler(ocrService),
		AI:        v1.NewAIHandler(aiService),
		Question:  v1.NewQuestionHandler(questionService),
		Mobile:    v1.NewMobileHandler(mobileService),
	}

	return httpx.NewRouter(appLogger, authService, handlers), nil
}
