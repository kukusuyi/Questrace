package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/kukusuyi/Questrace/backend/internal/domain/dto"
	ai "github.com/kukusuyi/Questrace/backend/internal/infra/ai"
)

type subjectRoute struct {
	SubjectID        string `json:"subject_id"`
	CourseID         string `json:"course_id"`
	Chapter          string `json:"chapter"`
	SuggestedSubject string `json:"suggested_subject"`
	Conflict         string `json:"conflict"`
}

func (s *AIService) classify(ctx context.Context, provider ai.ProviderClient, req dto.AnalyzeWrongQuestionRequest) (dto.AnalyzeWrongQuestionRequest, error) {
	items, err := s.Subjects.List(ctx)
	if err != nil {
		return req, err
	}
	if req.SubjectID != "" {
		if _, err = s.Subjects.Resolve(ctx, req.SubjectID, req.CourseID, req.Chapter); err != nil {
			return req, err
		}
	}
	stage, err := s.Subjects.EducationStage(ctx)
	if err != nil {
		return req, err
	}
	items = SubjectsForStage(items, stage, req.SubjectID)
	payload, _ := json.Marshal(map[string]any{"education_stage": stage, "available_subjects": items, "selected_subject_id": req.SubjectID, "selected_course_id": req.CourseID, "selected_chapter": req.Chapter, "question_json": req.QuestionJSON})
	completion, err := provider.ChatCompletion(ctx, ai.CompletionRequest{Model: req.ModelName, Messages: []ai.Message{
		{Role: "system", Content: `你是多学科错题分类助手。只根据题目内容识别学科、课程和章节，不解题。
education_stage是用户当前阶段：university大学，highschool高中；人工选择优先。自动识别参考此阶段，不能仅因有公式就把材料、机械、医学、经济学等专业内容归入数学。学科范围开放，available_subjects只是已有目录，绝不代表所有学科。
已有学科从 available_subjects 返回准确ID，不得创造ID。其他专业返回空subject_id，并用suggested_subject给出准确的学科名称（如材料力学、机械原理、材料科学基础）。只涉及高中基础数学且用户阶段为高中时归高中数学；大学阶段仍需判断是否真是考研数学。408仅适用于对应计算机四门课程。
人工选择优先。如内容与人工选择冲突，在conflict中简短说明，仍保留选择。信息不足时subject_id为空；无法区分高中与考研数学时不要猜测。
408课程从该学科courses选择。专用学科章节只能从对应列表选择，不确定则留空。通用学科可用简短章节描述。
未列出学科时返回空subject_id，suggested_subject给出建议名称。仅输出严格JSON：{"subject_id":"","course_id":"","chapter":"","suggested_subject":"","conflict":""}。`},
		{Role: "user", Content: string(payload)},
	}})
	if err != nil {
		return req, err
	}
	var route subjectRoute
	if err = json.Unmarshal([]byte(stripAnalyzeMarkdownJSON(strings.TrimSpace(completion.Content))), &route); err != nil {
		return req, &analyzeResultParseError{Err: err}
	}
	if req.SubjectID == "" {
		req.SubjectID = strings.TrimSpace(route.SubjectID)
		req.CourseID = strings.TrimSpace(route.CourseID)
	} else if req.CourseID == "" && route.SubjectID == req.SubjectID {
		req.CourseID = strings.TrimSpace(route.CourseID)
	}
	req.SuggestedSubject = route.SuggestedSubject
	req.Warnings = []string{}
	if route.Conflict != "" {
		req.Warnings = append(req.Warnings, route.Conflict)
	}
	if req.SubjectID == "" {
		req.Subject = "待分类"
		req.CourseID = ""
		req.Chapter = ""
		return req, nil
	}
	x, err := s.Subjects.Resolve(ctx, req.SubjectID, req.CourseID, "")
	if err != nil {
		return req, &analyzeResultParseError{Err: err}
	}
	req.Subject = x.Name
	if req.Chapter == "" && route.SubjectID == req.SubjectID && route.CourseID == req.CourseID {
		if _, err = s.Subjects.Resolve(ctx, req.SubjectID, req.CourseID, route.Chapter); err == nil {
			req.Chapter = route.Chapter
		}
	}
	return req, nil
}
func scopedPrompts(subject, course string) ([]analyzeChapterPrompt, error) {
	if subject == "math_grad" {
		return discoverAnalyzeChapterPrompts()
	}
	if subject != "cs408" || course == "" {
		return nil, nil
	}
	valid := []string{"data_structures", "computer_organization", "operating_systems", "computer_networks"}
	if !slices.Contains(valid, course) {
		return nil, fmt.Errorf("invalid course")
	}
	dir := "prompts/cs408/" + course
	entries, err := fs.ReadDir(promptFiles, dir)
	if err != nil {
		return nil, err
	}
	out := []analyzeChapterPrompt{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		b, err := fs.ReadFile(promptFiles, dir+"/"+e.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, analyzeChapterPrompt{Name: strings.TrimSuffix(e.Name(), ".md"), Content: string(b), Path: dir + "/" + e.Name()})
	}
	return out, nil
}
func generalPrompt() (analyzeChapterPrompt, error) {
	b, err := fs.ReadFile(promptFiles, "prompts/general.md")
	return analyzeChapterPrompt{Content: string(b)}, err
}
func constrainTags(result *llmAnalyzeResult, prompt string) {
	allowed := map[string]map[string]bool{}
	section := ""
	for _, line := range strings.Split(prompt, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "## ") {
			section = ""
		}
		if strings.HasPrefix(line, "#") {
			for _, key := range []string{"knowledge_points", "problem_type", "method", "mistake_reason"} {
				if strings.Contains(line, key) {
					section = key
					if allowed[key] == nil {
						allowed[key] = map[string]bool{}
					}
					break
				}
			}
		}
		if section != "" && strings.HasPrefix(line, "- ") {
			allowed[section][strings.TrimSpace(strings.TrimPrefix(line, "- "))] = true
		}
	}
	keep := func(key string, values []string) []string {
		out := []string{}
		for _, v := range values {
			if allowed[key][v] {
				out = append(out, v)
			}
		}
		return out
	}
	result.Tags.KnowledgePoints = keep("knowledge_points", result.Tags.KnowledgePoints)
	result.Tags.ProblemType = keep("problem_type", result.Tags.ProblemType)
	result.Tags.Method = keep("method", result.Tags.Method)
	result.Tags.MistakeReason = keep("mistake_reason", result.Tags.MistakeReason)
	if len(result.Tags.Method) > 3 {
		result.Tags.Method = result.Tags.Method[:3]
	}
}
