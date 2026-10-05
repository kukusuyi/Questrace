package http

import (
	"log/slog"
	"net/http"

	"github.com/kukusuyi/Questrace/backend/internal/http/handler"
	v1 "github.com/kukusuyi/Questrace/backend/internal/http/handler/v1"
	"github.com/kukusuyi/Questrace/backend/internal/http/middleware"
	"github.com/kukusuyi/Questrace/backend/internal/openapi"
	"github.com/kukusuyi/Questrace/backend/internal/service"
)

type V1Handlers struct {
	Auth      *v1.AuthHandler
	User      *v1.UserHandler
	Dashboard *v1.DashboardHandler
	Tag       *v1.TagHandler
	File      *v1.FileHandler
	OCR       *v1.OCRHandler
	AI        *v1.AIHandler
	Question  *v1.QuestionHandler
	Mobile    *v1.MobileHandler
}

func NewRouter(appLogger *slog.Logger, authService *service.AuthService, handlers V1Handlers) http.Handler {
	mux := http.NewServeMux()
	docsHandler := handler.NewDocsHandler(openapi.BuildSpec())

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /docs", docsHandler.DocsPage)
	mux.HandleFunc("GET /docs/openapi.json", docsHandler.OpenAPIJSON)

	mux.HandleFunc("POST /api/v1/auth/register", handlers.Auth.Register)
	mux.HandleFunc("POST /api/v1/auth/login", handlers.Auth.Login)

	mux.HandleFunc("GET /api/v1/users/me", handlers.User.GetMe)
	mux.HandleFunc("PUT /api/v1/users/me", handlers.User.UpdateMe)
	mux.HandleFunc("GET /api/v1/dashboard/summary", handlers.Dashboard.Summary)
	mux.HandleFunc("GET /api/v1/dashboard/recent", handlers.Dashboard.Recent)
	mux.HandleFunc("GET /api/v1/dashboard/tags", handlers.Dashboard.Tags)
	mux.HandleFunc("GET /api/v1/tags", handlers.Tag.List)
	mux.HandleFunc("POST /api/v1/tags", handlers.Tag.Create)
	mux.HandleFunc("DELETE /api/v1/tags/{tagID}", handlers.Tag.Delete)
	mux.HandleFunc("POST /api/v1/files/images", handlers.File.UploadImage)
	mux.HandleFunc("POST /api/v1/ocr/wrong-question-json", handlers.OCR.RecognizeWrongQuestion)
	mux.HandleFunc("GET /api/v1/ai/model-providers", handlers.AI.ListProviders)
	mux.HandleFunc("GET /api/v1/ai/model-providers/{providerName}/models", handlers.AI.ListProviderModels)
	mux.HandleFunc("GET /api/v1/subjects", handlers.AI.ListSubjects)
	mux.HandleFunc("POST /api/v1/subjects", handlers.AI.CreateSubject)
	mux.HandleFunc("POST /api/v1/wrong-questions/classification", handlers.Question.Reclassify)
	mux.HandleFunc("GET /api/v1/ai/chapters", handlers.AI.ListChapters)
	mux.HandleFunc("POST /api/v1/ai/analyze-wrong-question", handlers.AI.AnalyzeWrongQuestion)
	mux.HandleFunc("POST /api/v1/wrong-questions", handlers.Question.Create)
	mux.HandleFunc("GET /api/v1/wrong-questions", handlers.Question.List)
	mux.HandleFunc("GET /api/v1/wrong-questions/export/print", handlers.Question.ExportPrint)
	mux.HandleFunc("POST /api/v1/wrong-questions/similar-by-json", handlers.Question.SimilarByJSON)
	mux.HandleFunc("GET /api/v1/wrong-questions/{questionID}", handlers.Question.Detail)
	mux.HandleFunc("PUT /api/v1/wrong-questions/{questionID}", handlers.Question.Update)
	mux.HandleFunc("DELETE /api/v1/wrong-questions/{questionID}", handlers.Question.Delete)
	mux.HandleFunc("POST /api/v1/wrong-questions/{questionID}/similar", handlers.Question.Similar)

	mux.HandleFunc("GET /api/v1/mobile/latest-version", handlers.Mobile.GetLatestVersion)

	return middleware.Chain(
		mux,
		middleware.Recovery(appLogger),
		middleware.CORS(),
		middleware.Auth(authService),
	)
}
