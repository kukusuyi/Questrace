package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/kukusuyi/Questrace/backend/internal/domain/dto"
	"github.com/kukusuyi/Questrace/backend/internal/domain/model"
	aiclient "github.com/kukusuyi/Questrace/backend/internal/infra/ai"
	apperrors "github.com/kukusuyi/Questrace/backend/internal/pkg/errors"
	"github.com/kukusuyi/Questrace/backend/internal/pkg/validator"
	"github.com/kukusuyi/Questrace/backend/internal/repository"
)

const (
	analysisTypeAnalyzeWrongQuestion   = "analyze_wrong_question"
	chapterRouterPromptRelativePath    = "prompts/chapter_router.md"
	analyzeChapterPromptDirRelativeDir = "prompts/chapters"
)

var errInvalidAnalyzeChapterRoute = errors.New("invalid analyze chapter route")
var errUnknownAnalyzeChapter = errors.New("unknown analyze chapter")

type AIService struct {
	Subjects                *SubjectService
	registry                *aiclient.Registry
	recordRepo              repository.AIAnalysisRecordRepository
	chapterRouterPromptText string
	defaultProviderName     string
	defaultProviderModel    string
}

type llmAnalyzeResult struct {
	Chapter         string        `json:"chapter,omitempty"`
	Tags            dto.TagGroups `json:"tags"`
	SemanticSummary string        `json:"semantic_summary"`
	MistakeSummary  string        `json:"mistake_summary"`
}

type llmAnalyzeChapterRouteResult struct {
	Chapter string `json:"chapter"`
}

type analyzeChapterPrompt struct {
	Name    string
	Path    string
	Content string
}

func NewAIService(
	registry *aiclient.Registry,
	recordRepo repository.AIAnalysisRecordRepository,
	defaultProviderName string,
	defaultProviderModel string,
) (*AIService, error) {
	chapterRouterPromptText, err := loadChapterRouterPrompt()
	if err != nil {
		return nil, err
	}
	if _, err := discoverAnalyzeChapterPrompts(); err != nil {
		return nil, err
	}

	return &AIService{
		registry:                registry,
		recordRepo:              recordRepo,
		chapterRouterPromptText: chapterRouterPromptText,
		defaultProviderName:     strings.TrimSpace(defaultProviderName),
		defaultProviderModel:    strings.TrimSpace(defaultProviderModel),
	}, nil
}

func (s *AIService) ListProviders() dto.AIProviderListResponse {
	configs := s.registry.Providers()
	items := make([]dto.AIProviderItem, 0, len(configs))
	for _, item := range configs {
		items = append(items, dto.AIProviderItem{
			ProviderName:    item.Name,
			ProviderType:    item.ProviderType,
			ConfiguredModel: item.Model,
		})
	}

	return dto.AIProviderListResponse{List: items}
}

func (s *AIService) ListProviderModels(ctx context.Context, providerName string) (dto.AIProviderModelListResponse, error) {
	providerName = strings.TrimSpace(providerName)
	if providerName == "" {
		return dto.AIProviderModelListResponse{}, apperrors.New(http.StatusBadRequest, 40001, "provider_name 不能为空")
	}

	provider, ok := s.registry.Provider(providerName)
	if !ok {
		return dto.AIProviderModelListResponse{}, apperrors.New(http.StatusNotFound, 40405, "模型厂商不存在")
	}

	models, err := provider.ListModels(ctx)
	if err != nil {
		return dto.AIProviderModelListResponse{}, apperrors.New(http.StatusBadGateway, 50004, apperrors.ProviderMessage(0, err.Error()))
	}

	items := make([]dto.AIProviderModelItem, 0, len(models))
	for _, item := range models {
		items = append(items, dto.AIProviderModelItem{ModelName: item.ID})
	}

	return dto.AIProviderModelListResponse{
		ProviderName: providerName,
		List:         items,
	}, nil
}

func (s *AIService) ListChapters() (dto.AIChapterListResponse, error) {
	chapterPrompts, err := discoverAnalyzeChapterPrompts()
	if err != nil {
		return dto.AIChapterListResponse{}, err
	}

	return dto.AIChapterListResponse{
		List: chapterPromptNames(chapterPrompts),
	}, nil
}

func (s *AIService) Analyze(ctx context.Context, req dto.AnalyzeWrongQuestionRequest) (dto.AnalyzeWrongQuestionResponse, error) {
	if err := validator.RequireString(req.QuestionJSON.QuestionCore, "question_json.question_core"); err != nil {
		return dto.AnalyzeWrongQuestionResponse{}, err
	}

	resolvedReq, err := s.resolveAnalyzeRequest(req)
	if err != nil {
		return dto.AnalyzeWrongQuestionResponse{}, err
	}

	userID, ok := GetUserID(ctx)
	if !ok {
		return dto.AnalyzeWrongQuestionResponse{}, apperrors.New(http.StatusUnauthorized, 40100, "未获取到用户信息")
	}

	inputQuestionJSONBytes, err := json.Marshal(req.QuestionJSON)
	if err != nil {
		return dto.AnalyzeWrongQuestionResponse{}, fmt.Errorf("marshal question_json: %w", err)
	}

	provider, ok := s.registry.Provider(resolvedReq.ProviderName)
	if !ok {
		return dto.AnalyzeWrongQuestionResponse{}, apperrors.New(http.StatusNotFound, 40405, "模型厂商不存在")
	}

	resolvedReq, err = s.classify(ctx, provider, resolvedReq)
	if err != nil {
		return dto.AnalyzeWrongQuestionResponse{}, apperrors.New(502, 50006, "学科识别失败，请重试或手动整理："+err.Error())
	}
	result, err := s.analyzeWithProvider(ctx, provider, resolvedReq)
	if err != nil {
		s.recordFailure(userID, resolvedReq, inputQuestionJSONBytes, apperrors.ProviderMessage(0, err.Error()))
		if errors.Is(err, errUnknownAnalyzeChapter) {
			return dto.AnalyzeWrongQuestionResponse{}, apperrors.New(http.StatusBadRequest, 40001, "chapter 不存在于本地章节提示词目录")
		}
		if errors.Is(err, errInvalidAnalyzeChapterRoute) {
			return dto.AnalyzeWrongQuestionResponse{}, apperrors.New(http.StatusBadGateway, 50006, "模型返回的章节信息格式异常，请重试或手动选择章节。")
		}
		var analyzeParseError *analyzeResultParseError
		if errors.As(err, &analyzeParseError) {
			return dto.AnalyzeWrongQuestionResponse{}, apperrors.New(http.StatusBadGateway, 50006, "模型返回格式异常，请重试或更换兼容模型，也可以手动整理。")
		}
		var promptLoadError *analyzePromptLoadError
		if errors.As(err, &promptLoadError) {
			return dto.AnalyzeWrongQuestionResponse{}, apperrors.New(http.StatusInternalServerError, 50007, "分析提示词无法读取，请重新启动或重新安装电脑端程序。")
		}
		return dto.AnalyzeWrongQuestionResponse{}, apperrors.New(http.StatusBadGateway, 50005, apperrors.ProviderMessage(0, err.Error()))
	}

	if err := s.recordSuccess(userID, resolvedReq, inputQuestionJSONBytes, result); err != nil {
		return dto.AnalyzeWrongQuestionResponse{}, err
	}

	status := "pending"
	if resolvedReq.SubjectID != "" {
		status = "confirmed"
	}
	return dto.AnalyzeWrongQuestionResponse{
		SubjectID: resolvedReq.SubjectID, Subject: resolvedReq.Subject, CourseID: resolvedReq.CourseID, ClassificationStatus: status, SuggestedSubject: resolvedReq.SuggestedSubject, Warnings: resolvedReq.Warnings,
		Chapter:         result.Chapter,
		Tags:            result.Tags,
		SemanticSummary: result.SemanticSummary,
		MistakeSummary:  result.MistakeSummary,
	}, nil
}

func (s *AIService) analyzeWithProvider(ctx context.Context, provider aiclient.ProviderClient, req dto.AnalyzeWrongQuestionRequest) (llmAnalyzeResult, error) {
	chapterPrompts, err := scopedPrompts(req.SubjectID, req.CourseID)
	if err != nil {
		return llmAnalyzeResult{}, &analyzePromptLoadError{Err: err}
	}
	var selectedPrompt analyzeChapterPrompt
	if len(chapterPrompts) > 0 {
		selectedPrompt, err = s.selectAnalyzeChapter(ctx, provider, req, s.currentChapterRouterPrompt(), chapterPrompts)
	} else {
		selectedPrompt, err = generalPrompt()
	}
	if err != nil {
		return llmAnalyzeResult{}, err
	}
	if selectedPrompt.Content == "" {
		selectedPrompt, err = generalPrompt()
		if err != nil {
			return llmAnalyzeResult{}, err
		}
	}

	payloadBytes, err := json.Marshal(buildAnalyzePayload(req))
	if err != nil {
		return llmAnalyzeResult{}, fmt.Errorf("marshal analyze payload: %w", err)
	}

	completion, err := provider.ChatCompletion(ctx, aiclient.CompletionRequest{
		Model: req.ModelName,
		Messages: []aiclient.Message{
			{
				Role:    "system",
				Content: buildAnalyzeSystemPrompt(selectedPrompt.Content),
			},
			{
				Role:    "user",
				Content: string(payloadBytes),
			},
		},
	})
	if err != nil {
		return llmAnalyzeResult{}, err
	}

	result, err := parseAnalyzeResult(completion.Content, req.QuestionJSON.WrongSolution)
	if err != nil {
		return llmAnalyzeResult{}, &analyzeResultParseError{Err: err}
	}
	if selectedPrompt.Name == "" && len(chapterPrompts) == 0 && req.Chapter != "" {
		result.Chapter = req.Chapter
	}
	if selectedPrompt.Name != "" {
		result.Chapter = selectedPrompt.Name
		constrainTags(&result, selectedPrompt.Content)
	} else if len(chapterPrompts) > 0 {
		result.Chapter = ""
	}
	if len(result.Tags.Method) > 3 {
		result.Tags.Method = result.Tags.Method[:3]
	}

	return result, nil
}

func (s *AIService) selectAnalyzeChapter(
	ctx context.Context,
	provider aiclient.ProviderClient,
	req dto.AnalyzeWrongQuestionRequest,
	chapterRouterPrompt string,
	chapterPrompts []analyzeChapterPrompt,
) (analyzeChapterPrompt, error) {
	manualChapter := strings.TrimSpace(req.Chapter)
	if manualChapter == "" {
		return s.routeAnalyzeChapter(ctx, provider, req, chapterRouterPrompt, chapterPrompts)
	}

	selectedPrompt, ok := findAnalyzeChapterPrompt(chapterPrompts, manualChapter)
	if !ok {
		return analyzeChapterPrompt{}, fmt.Errorf("%w: %s", errUnknownAnalyzeChapter, manualChapter)
	}

	return selectedPrompt, nil
}

func (s *AIService) routeAnalyzeChapter(
	ctx context.Context,
	provider aiclient.ProviderClient,
	req dto.AnalyzeWrongQuestionRequest,
	chapterRouterPrompt string,
	chapterPrompts []analyzeChapterPrompt,
) (analyzeChapterPrompt, error) {
	payloadBytes, err := json.Marshal(buildAnalyzeChapterRoutePayload(req, chapterPrompts))
	if err != nil {
		return analyzeChapterPrompt{}, fmt.Errorf("marshal analyze chapter route payload: %w", err)
	}

	completion, err := provider.ChatCompletion(ctx, aiclient.CompletionRequest{
		Model: req.ModelName,
		Messages: []aiclient.Message{
			{
				Role:    "system",
				Content: buildAnalyzeChapterRouteSystemPrompt(chapterRouterPrompt, chapterPrompts),
			},
			{
				Role:    "user",
				Content: string(payloadBytes),
			},
		},
	})
	if err != nil {
		return analyzeChapterPrompt{}, err
	}

	chapterName, err := parseAnalyzeChapterRouteResult(completion.Content)
	if err != nil {
		return analyzeChapterPrompt{}, fmt.Errorf("%w: %v", errInvalidAnalyzeChapterRoute, err)
	}

	if chapterName == "" {
		return analyzeChapterPrompt{}, nil
	}
	for _, chapterPrompt := range chapterPrompts {
		if chapterPrompt.Name == chapterName {
			return chapterPrompt, nil
		}
	}

	return analyzeChapterPrompt{}, fmt.Errorf("%w: chapter %q not found in local prompt directory", errInvalidAnalyzeChapterRoute, chapterName)
}

func (s *AIService) loadCurrentAnalyzePrompts() (string, []analyzeChapterPrompt, error) {
	chapterPrompts, err := discoverAnalyzeChapterPrompts()
	if err != nil {
		return "", nil, err
	}

	chapterRouterPrompt := s.currentChapterRouterPrompt()
	if strings.TrimSpace(chapterRouterPrompt) == "" {
		return "", nil, fmt.Errorf("chapter router prompt is empty")
	}

	return chapterRouterPrompt, chapterPrompts, nil
}

func (s *AIService) currentChapterRouterPrompt() string {
	promptText, err := loadChapterRouterPrompt()
	if err == nil && strings.TrimSpace(promptText) != "" {
		return promptText
	}
	return s.chapterRouterPromptText
}

func (s *AIService) resolveAnalyzeRequest(req dto.AnalyzeWrongQuestionRequest) (dto.AnalyzeWrongQuestionRequest, error) {
	req.ProviderName = strings.TrimSpace(req.ProviderName)
	req.ModelName = strings.TrimSpace(req.ModelName)
	req.Chapter = strings.TrimSpace(req.Chapter)

	if req.ProviderName == "" {
		req.ProviderName = s.defaultProviderName
	}
	if req.ModelName == "" {
		req.ModelName = s.defaultProviderModel
	}

	if err := validator.RequireString(req.ProviderName, "provider_name"); err != nil {
		return dto.AnalyzeWrongQuestionRequest{}, err
	}
	if err := validator.RequireString(req.ModelName, "model_name"); err != nil {
		return dto.AnalyzeWrongQuestionRequest{}, err
	}

	return req, nil
}

func (s *AIService) recordSuccess(userID int64, req dto.AnalyzeWrongQuestionRequest, inputPayload []byte, result llmAnalyzeResult) error {
	if s.recordRepo == nil {
		return nil
	}

	tagsJSON, err := json.Marshal(result.Tags)
	if err != nil {
		return fmt.Errorf("marshal ai tags: %w", err)
	}

	_, err = s.recordRepo.Create(model.AIAnalysisRecord{
		UserID:            userID,
		ProviderName:      strings.TrimSpace(req.ProviderName),
		ModelName:         strings.TrimSpace(req.ModelName),
		AnalysisType:      analysisTypeAnalyzeWrongQuestion,
		InputQuestionJSON: string(inputPayload),
		OutputTagsJSON:    string(tagsJSON),
		SemanticSummary:   result.SemanticSummary,
		MistakeSummary:    result.MistakeSummary,
		Status:            "success",
	})
	if err != nil {
		return fmt.Errorf("create ai_analysis_record: %w", err)
	}

	return nil
}

func (s *AIService) recordFailure(userID int64, req dto.AnalyzeWrongQuestionRequest, inputPayload []byte, errorMessage string) {
	if s.recordRepo == nil {
		return
	}

	_, _ = s.recordRepo.Create(model.AIAnalysisRecord{
		UserID:            userID,
		ProviderName:      strings.TrimSpace(req.ProviderName),
		ModelName:         strings.TrimSpace(req.ModelName),
		AnalysisType:      analysisTypeAnalyzeWrongQuestion,
		InputQuestionJSON: string(inputPayload),
		Status:            "failed",
		ErrorMessage:      errorMessage,
	})
}

func buildAnalyzePayload(req dto.AnalyzeWrongQuestionRequest) map[string]any {
	payload := map[string]any{
		"question_json": req.QuestionJSON,
		"subject_id":    req.SubjectID, "subject": req.Subject, "course_id": req.CourseID,
	}
	if strings.TrimSpace(req.Chapter) != "" {
		payload["chapter"] = strings.TrimSpace(req.Chapter)
	}
	if req.OCRContext != nil {
		payload["ocr_context"] = req.OCRContext
	}
	return payload
}

func buildAnalyzeChapterRoutePayload(req dto.AnalyzeWrongQuestionRequest, chapterPrompts []analyzeChapterPrompt) map[string]any {
	payload := buildAnalyzePayload(req)
	payload["available_chapters"] = chapterPromptNames(chapterPrompts)
	return payload
}

func buildAnalyzeChapterRouteSystemPrompt(basePrompt string, chapterPrompts []analyzeChapterPrompt) string {
	return strings.TrimSpace(basePrompt) + `

当前学科可选章节列表（信息不足可返回空）：
` + formatChapterPromptNames(chapterPrompts) + `

你现在执行的是“错题章节识别接口”，输入一定是 JSON。

你必须结合题目内容与可选章节列表，输出以下严格合法 JSON：
{
  "chapter": ""
}

附加规则：
1. chapter 必须与可选章节列表中的某一项完全一致，不允许改写，不允许输出列表外的章节名。
2. 如果题目跨多个章节，选择最核心、最直接对应的主章节。
3. 如果信息不足，chapter 返回空字符串，不要猜测。
4. 只输出 JSON，不要输出 Markdown，不要输出解释。`
}

func buildAnalyzeSystemPrompt(basePrompt string) string {
	return strings.TrimSpace(basePrompt) + `

你现在执行的是“错题 AI 分析接口”，输入一定是 JSON。

你必须在遵守上述标签规则的前提下，输出以下严格合法 JSON：
{
  "chapter": "",
  "tags": {
    "knowledge_points": [],
    "problem_type": [],
    "method": [],
    "mistake_reason": []
  },
  "semantic_summary": "",
  "mistake_summary": ""
}

附加规则：
1. semantic_summary 必须是 1 到 2 句中文摘要，概括题目考察内容与结构，不要照抄原题全文。
2. 如果 wrong_solution 为空，mistake_summary 应输出空字符串。
3. 如果 wrong_solution 非空，mistake_summary 应用中文概括错误思路或错误步骤，不要直接复制 wrong_solution。
4. 所有字段必须存在，数组为空时返回 []。
5. 只输出 JSON，不要输出 Markdown，不要输出解释。`
}

func parseAnalyzeChapterRouteResult(raw string) (string, error) {
	cleaned := stripAnalyzeMarkdownJSON(strings.TrimSpace(raw))

	var result llmAnalyzeChapterRouteResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return "", err
	}

	chapter := strings.TrimSpace(result.Chapter)

	return chapter, nil
}

func parseAnalyzeResult(raw string, wrongSolution string) (llmAnalyzeResult, error) {
	cleaned := stripAnalyzeMarkdownJSON(strings.TrimSpace(raw))

	var result llmAnalyzeResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return llmAnalyzeResult{}, err
	}

	result.Tags = normalizeAnalyzeTagGroups(result.Tags)
	result.SemanticSummary = strings.TrimSpace(result.SemanticSummary)
	result.MistakeSummary = strings.TrimSpace(result.MistakeSummary)

	if result.SemanticSummary == "" {
		return llmAnalyzeResult{}, fmt.Errorf("semantic_summary 不能为空")
	}
	if strings.TrimSpace(wrongSolution) == "" {
		result.MistakeSummary = ""
		result.Tags.MistakeReason = []string{}
	}

	return result, nil
}

func normalizeAnalyzeTagGroups(tags dto.TagGroups) dto.TagGroups {
	return dto.TagGroups{
		KnowledgePoints: normalizeAnalyzeStringSlice(tags.KnowledgePoints),
		ProblemType:     normalizeAnalyzeStringSlice(tags.ProblemType),
		Method:          normalizeAnalyzeStringSlice(tags.Method),
		MistakeReason:   normalizeAnalyzeStringSlice(tags.MistakeReason),
	}
}

func normalizeAnalyzeStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	if len(result) == 0 {
		return []string{}
	}

	return result
}

func stripAnalyzeMarkdownJSON(s string) string {
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimSpace(s)
	}
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSpace(s)
	}
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	return s
}

func loadChapterRouterPrompt() (string, error) {
	data, err := fs.ReadFile(promptFiles, "prompts/chapter_router.md")
	return string(data), err
}

func discoverAnalyzeChapterPrompts() ([]analyzeChapterPrompt, error) {
	entries, err := fs.ReadDir(promptFiles, "prompts/chapters")
	if err != nil {
		return nil, err
	}
	var result []analyzeChapterPrompt
	for _, e := range entries {
		data, err := fs.ReadFile(promptFiles, "prompts/chapters/"+e.Name())
		if err != nil {
			return nil, err
		}
		result = append(result, analyzeChapterPrompt{Name: strings.TrimSuffix(e.Name(), ".md"), Path: e.Name(), Content: string(data)})
	}
	return result, nil
}

func loadAnalyzeChapterPromptsFromDir(dir string) ([]analyzeChapterPrompt, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	prompts := make([]analyzeChapterPrompt, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			continue
		}

		name := strings.TrimSpace(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
		if name == "" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read chapter prompt %s: %w", path, err)
		}

		content := strings.TrimSpace(string(data))
		if content == "" {
			continue
		}

		prompts = append(prompts, analyzeChapterPrompt{
			Name:    name,
			Path:    path,
			Content: string(data),
		})
	}

	if len(prompts) == 0 {
		return nil, fmt.Errorf("no chapter prompt files found in %s", dir)
	}

	sort.Slice(prompts, func(i, j int) bool {
		return strings.Compare(prompts[i].Name, prompts[j].Name) < 0
	})

	return prompts, nil
}

func chapterRouterPromptPathCandidates() []string {
	candidates := []string{
		chapterRouterPromptRelativePath,
		filepath.Join("..", chapterRouterPromptRelativePath),
		filepath.Join("..", "..", chapterRouterPromptRelativePath),
	}

	if _, currentFile, _, ok := runtime.Caller(0); ok {
		candidates = append(candidates, filepath.Join(filepath.Dir(currentFile), "..", "..", chapterRouterPromptRelativePath))
	}

	return uniqueCleanPaths(candidates)
}

func analyzeChapterPromptDirCandidates() []string {
	candidates := []string{
		analyzeChapterPromptDirRelativeDir,
		filepath.Join("..", analyzeChapterPromptDirRelativeDir),
		filepath.Join("..", "..", analyzeChapterPromptDirRelativeDir),
	}

	if _, currentFile, _, ok := runtime.Caller(0); ok {
		candidates = append(candidates, filepath.Join(filepath.Dir(currentFile), "..", "..", analyzeChapterPromptDirRelativeDir))
	}

	return uniqueCleanPaths(candidates)
}

func uniqueCleanPaths(candidates []string) []string {
	seen := make(map[string]struct{}, len(candidates))
	unique := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		unique = append(unique, candidate)
	}
	return unique
}

func chapterPromptNames(chapterPrompts []analyzeChapterPrompt) []string {
	names := make([]string, 0, len(chapterPrompts))
	for _, chapterPrompt := range chapterPrompts {
		names = append(names, chapterPrompt.Name)
	}
	return names
}

func findAnalyzeChapterPrompt(chapterPrompts []analyzeChapterPrompt, chapterName string) (analyzeChapterPrompt, bool) {
	for _, chapterPrompt := range chapterPrompts {
		if chapterPrompt.Name == chapterName {
			return chapterPrompt, true
		}
	}

	return analyzeChapterPrompt{}, false
}

func formatChapterPromptNames(chapterPrompts []analyzeChapterPrompt) string {
	names := chapterPromptNames(chapterPrompts)
	lines := make([]string, 0, len(names))
	for _, name := range names {
		lines = append(lines, "- "+name)
	}
	return strings.Join(lines, "\n")
}

type analyzePromptLoadError struct {
	Err error
}

func (e *analyzePromptLoadError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *analyzePromptLoadError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type analyzeResultParseError struct {
	Err error
}

func (e *analyzeResultParseError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *analyzeResultParseError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
