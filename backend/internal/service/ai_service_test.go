package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kukusuyi/Questrace/backend/internal/config"
	"github.com/kukusuyi/Questrace/backend/internal/domain/dto"
	aiclient "github.com/kukusuyi/Questrace/backend/internal/infra/ai"
)

type stubAIProvider struct {
	responses []string
	requests  []aiclient.CompletionRequest
}

func (p *stubAIProvider) ChatCompletion(_ context.Context, req aiclient.CompletionRequest) (aiclient.CompletionResponse, error) {
	p.requests = append(p.requests, req)
	if len(p.responses) == 0 {
		return aiclient.CompletionResponse{}, errors.New("no stub response configured")
	}

	response := p.responses[0]
	p.responses = p.responses[1:]
	return aiclient.CompletionResponse{Content: response}, nil
}

func (p *stubAIProvider) ListModels(context.Context) ([]aiclient.ProviderModel, error) {
	return nil, nil
}

func (p *stubAIProvider) Config() config.AIModelConfig {
	return config.AIModelConfig{Name: "stub", Model: "stub-model"}
}

func TestAIServiceAnalyzeWithProviderUsesDynamicallyDiscoveredChapterPrompt(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	tempDir := t.TempDir()
	originalFS := promptFiles
	promptFiles = os.DirFS(tempDir)
	t.Cleanup(func() { promptFiles = originalFS })
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tempDir, "prompts", "chapters"), 0o755); err != nil {
		t.Fatalf("mkdir prompts dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "prompts", "chapter_router.md"), []byte("router-prompt-v1"), 0o644); err != nil {
		t.Fatalf("write chapter router prompt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "prompts", "chapters", "函数的极限和连续.md"), []byte("limit-prompt"), 0o644); err != nil {
		t.Fatalf("write limit prompt: %v", err)
	}

	service, err := NewAIService(nil, nil, "stub", "stub-model")
	if err != nil {
		t.Fatalf("new ai service: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, "prompts", "chapters", "定积分.md"), []byte("integral-prompt"), 0o644); err != nil {
		t.Fatalf("write integral prompt: %v", err)
	}

	provider := &stubAIProvider{
		responses: []string{
			`{"chapter":"定积分"}`,
			`{
				"tags": {
					"knowledge_points": ["原函数"],
					"problem_type": ["积分计算"],
					"method": ["换元"],
					"mistake_reason": []
				},
				"semantic_summary": "题目考查定积分计算与换元思路。",
				"mistake_summary": ""
			}`,
		},
	}

	result, err := service.analyzeWithProvider(context.Background(), provider, dto.AnalyzeWrongQuestionRequest{
		SubjectID:    "math_grad",
		ProviderName: "stub",
		ModelName:    "stub-model",
		QuestionJSON: dto.QuestionJSON{
			QuestionCore: `计算 \int_0^1 x^2 \, dx`,
		},
	})
	if err != nil {
		t.Fatalf("analyze with provider: %v", err)
	}

	if result.SemanticSummary != "题目考查定积分计算与换元思路。" {
		t.Fatalf("semantic summary = %q", result.SemanticSummary)
	}
	if result.Chapter != "定积分" {
		t.Fatalf("chapter = %q, want %q", result.Chapter, "定积分")
	}
	if len(provider.requests) != 2 {
		t.Fatalf("provider request count = %d, want 2", len(provider.requests))
	}
	if got := provider.requests[0].Messages[0].Content; !strings.Contains(got, "router-prompt-v1") {
		t.Fatalf("route system prompt missing router prompt content: %q", got)
	}
	if got := provider.requests[1].Messages[0].Content; !strings.Contains(got, "integral-prompt") {
		t.Fatalf("final system prompt missing selected chapter prompt: %q", got)
	}

	var routePayload struct {
		AvailableChapters []string `json:"available_chapters"`
	}
	if err := json.Unmarshal([]byte(provider.requests[0].Messages[1].Content), &routePayload); err != nil {
		t.Fatalf("unmarshal route payload: %v", err)
	}
	if len(routePayload.AvailableChapters) != 2 {
		t.Fatalf("available chapters count = %d, want 2", len(routePayload.AvailableChapters))
	}
	if !containsStringValue(routePayload.AvailableChapters, "函数的极限和连续") {
		t.Fatalf("available chapters missing 函数的极限和连续: %#v", routePayload.AvailableChapters)
	}
	if !containsStringValue(routePayload.AvailableChapters, "定积分") {
		t.Fatalf("available chapters missing 定积分: %#v", routePayload.AvailableChapters)
	}
}

func TestAIServiceAnalyzeWithProviderRejectsUnknownChapterRoute(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	tempDir := t.TempDir()
	originalFS := promptFiles
	promptFiles = os.DirFS(tempDir)
	t.Cleanup(func() { promptFiles = originalFS })
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tempDir, "prompts", "chapters"), 0o755); err != nil {
		t.Fatalf("mkdir prompts dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "prompts", "chapter_router.md"), []byte("router-prompt-v1"), 0o644); err != nil {
		t.Fatalf("write chapter router prompt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "prompts", "chapters", "函数的极限和连续.md"), []byte("limit-prompt"), 0o644); err != nil {
		t.Fatalf("write limit prompt: %v", err)
	}

	service, err := NewAIService(nil, nil, "stub", "stub-model")
	if err != nil {
		t.Fatalf("new ai service: %v", err)
	}

	provider := &stubAIProvider{
		responses: []string{
			`{"chapter":"概率统计"}`,
		},
	}

	_, err = service.analyzeWithProvider(context.Background(), provider, dto.AnalyzeWrongQuestionRequest{
		SubjectID:    "math_grad",
		ProviderName: "stub",
		ModelName:    "stub-model",
		QuestionJSON: dto.QuestionJSON{
			QuestionCore: `\lim_{x \to 0}\frac{\sin x}{x}`,
		},
	})
	if err == nil {
		t.Fatal("expected error for unknown chapter route")
	}
	if !errors.Is(err, errInvalidAnalyzeChapterRoute) {
		t.Fatalf("error should match errInvalidAnalyzeChapterRoute, got %v", err)
	}
	if len(provider.requests) != 1 {
		t.Fatalf("provider request count = %d, want 1", len(provider.requests))
	}
}

func TestAIServiceAnalyzeWithProviderSkipsRouteWhenChapterSpecified(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	tempDir := t.TempDir()
	originalFS := promptFiles
	promptFiles = os.DirFS(tempDir)
	t.Cleanup(func() { promptFiles = originalFS })
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tempDir, "prompts", "chapters"), 0o755); err != nil {
		t.Fatalf("mkdir prompts dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "prompts", "chapter_router.md"), []byte("router-prompt-v1"), 0o644); err != nil {
		t.Fatalf("write chapter router prompt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "prompts", "chapters", "定积分.md"), []byte("integral-prompt"), 0o644); err != nil {
		t.Fatalf("write integral prompt: %v", err)
	}

	service, err := NewAIService(nil, nil, "stub", "stub-model")
	if err != nil {
		t.Fatalf("new ai service: %v", err)
	}

	provider := &stubAIProvider{
		responses: []string{
			`{
				"tags": {
					"knowledge_points": ["定积分"],
					"problem_type": ["积分计算"],
					"method": ["换元"],
					"mistake_reason": []
				},
				"semantic_summary": "题目考查定积分计算。",
				"mistake_summary": ""
			}`,
		},
	}

	result, err := service.analyzeWithProvider(context.Background(), provider, dto.AnalyzeWrongQuestionRequest{
		SubjectID:    "math_grad",
		ProviderName: "stub",
		ModelName:    "stub-model",
		Chapter:      "定积分",
		QuestionJSON: dto.QuestionJSON{
			QuestionCore: `计算 \int_0^1 x^2 \, dx`,
		},
	})
	if err != nil {
		t.Fatalf("analyze with provider: %v", err)
	}

	if result.Chapter != "定积分" {
		t.Fatalf("chapter = %q, want %q", result.Chapter, "定积分")
	}
	if len(provider.requests) != 1 {
		t.Fatalf("provider request count = %d, want 1", len(provider.requests))
	}
	if got := provider.requests[0].Messages[0].Content; strings.Contains(got, "错题章节识别接口") {
		t.Fatalf("manual chapter analyze should skip route prompt: %q", got)
	}
}

func TestAIServiceListChapters(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	tempDir := t.TempDir()
	originalFS := promptFiles
	promptFiles = os.DirFS(tempDir)
	t.Cleanup(func() { promptFiles = originalFS })
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tempDir, "prompts", "chapters"), 0o755); err != nil {
		t.Fatalf("mkdir prompts dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "prompts", "chapter_router.md"), []byte("router-prompt-v1"), 0o644); err != nil {
		t.Fatalf("write chapter router prompt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "prompts", "chapters", "函数的极限和连续.md"), []byte("limit-prompt"), 0o644); err != nil {
		t.Fatalf("write limit prompt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "prompts", "chapters", "定积分.md"), []byte("integral-prompt"), 0o644); err != nil {
		t.Fatalf("write integral prompt: %v", err)
	}

	service, err := NewAIService(nil, nil, "stub", "stub-model")
	if err != nil {
		t.Fatalf("new ai service: %v", err)
	}

	resp, err := service.ListChapters()
	if err != nil {
		t.Fatalf("list chapters: %v", err)
	}
	if len(resp.List) != 2 {
		t.Fatalf("chapter count = %d, want 2", len(resp.List))
	}
	if !containsStringValue(resp.List, "函数的极限和连续") {
		t.Fatalf("missing chapter 函数的极限和连续: %#v", resp.List)
	}
	if !containsStringValue(resp.List, "定积分") {
		t.Fatalf("missing chapter 定积分: %#v", resp.List)
	}
}

func containsStringValue(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
