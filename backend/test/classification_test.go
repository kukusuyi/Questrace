package test

import (
	"fmt"
	"github.com/kukusuyi/Questrace/backend/internal/domain/dto"
	"net/http"
	"testing"
)

func TestClassificationAPIWorkflow(t *testing.T) {
	token := ensureToken(t)
	resp, body, err := doGet(baseURL+"/api/v1/subjects", token)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("catalog: %v %s", err, body)
	}
	var catalog struct {
		List []struct {
			ID string `json:"id"`
		}
	}
	catalog, err = unmarshalData[struct {
		List []struct {
			ID string `json:"id"`
		}
	}](body)
	if err != nil || len(catalog.List) != 7 {
		t.Fatalf("catalog: %v %s", err, body)
	}
	resp, body, err = doPost(baseURL+"/api/v1/subjects", token, map[string]string{"name": "天文学"})
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("custom: %v %s", err, body)
	}
	custom, _ := unmarshalData[struct {
		ID string `json:"id"`
	}](body)
	create := func(subject, course, chapter string) int64 {
		t.Helper()
		resp, body, err := doPost(baseURL+"/api/v1/wrong-questions", token, dto.CreateWrongQuestionRequest{SubjectID: subject, CourseID: course, Chapter: chapter, SourceType: "manual", QuestionJSON: dto.QuestionJSON{QuestionCore: "测试题目"}, SemanticSummary: "摘要", Tags: dto.TagGroups{KnowledgePoints: []string{"示例知识点"}}})
		if err != nil || resp.StatusCode != 200 {
			t.Fatalf("create: %v %s", err, body)
		}
		q, _ := unmarshalData[dto.CreateWrongQuestionResponse](body)
		return q.QuestionID
	}
	id := create("cs408", "operating_systems", "内存管理")
	create("highschool_math", "", "")
	create(custom.ID, "", "恒星")
	resp, body, err = doGet(baseURL+"/api/v1/wrong-questions?subject_id=cs408&course_id=operating_systems", token)
	if err != nil || resp.StatusCode != 200 {
		t.Fatal(err, string(body))
	}
	page, _ := unmarshalData[dto.PageResult[dto.QuestionListItem]](body)
	if len(page.List) != 1 || page.List[0].QuestionID != id {
		t.Fatalf("filter: %s", body)
	}
	req := map[string]any{"subject_id": "highschool_physics", "items": []map[string]any{{"question_id": id, "revision": page.List[0].Revision}}}
	resp, body, err = doPost(baseURL+"/api/v1/wrong-questions/classification", token, req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("reclassify: %v %s", err, body)
	}
	resp, body, err = doPost(baseURL+"/api/v1/wrong-questions/classification", token, req)
	if err != nil || resp.StatusCode != http.StatusConflict {
		t.Fatalf("conflict: %v %s", err, body)
	}
	_, body, _ = doGet(fmt.Sprintf("%s/api/v1/wrong-questions/%d", baseURL, id), token)
	q, err := unmarshalData[dto.QuestionDetail](body)
	if err != nil || q.SubjectID != "highschool_physics" || q.Subject != "高中物理" || q.CourseID != "" || q.Chapter != "" || !q.AnalysisStale || q.SemanticSummary != "摘要" {
		t.Fatalf("updated detail: %v %s", err, body)
	}
	resp, body, err = doGet(baseURL+"/api/v1/tags?subject_id=highschool_physics", token)
	if err != nil || resp.StatusCode != 200 {
		t.Fatal(err, string(body))
	}
	tags, _ := unmarshalData[dto.TagListResponse](body)
	if len(tags.List) != 0 {
		t.Fatalf("stale tags exposed: %s", body)
	}
	resp, body, err = doPost(baseURL+"/api/v1/wrong-questions/similar-by-json", token, map[string]any{"question_json": map[string]string{"question_core": "未分类题目"}, "use_tag_filter": false})
	if err != nil || resp.StatusCode != 409 {
		t.Fatalf("unclassified search: %v %s", err, body)
	}
}

func TestEducationStageAPIWorkflow(t *testing.T) {
	token, _ := registerFreshUser(t, "stage")
	resp, body, err := doGet(baseURL+"/api/v1/users/me", token)
	if err != nil || resp.StatusCode != 200 {
		t.Fatal(err, string(body))
	}
	me, err := unmarshalData[dto.UserMeResponse](body)
	if err != nil || me.EducationStage != "university" {
		t.Fatal(me, err)
	}
	resp, body, err = doPut(baseURL+"/api/v1/users/me", token, map[string]string{"education_stage": "highschool"})
	if err != nil || resp.StatusCode != 200 {
		t.Fatal(err, string(body))
	}
	me, err = unmarshalData[dto.UserMeResponse](body)
	if err != nil || me.EducationStage != "highschool" {
		t.Fatal(me, err)
	}
	resp, body, err = doGet(baseURL+"/api/v1/subjects", token)
	if err != nil || resp.StatusCode != 200 {
		t.Fatal(err, string(body))
	}
	catalog, err := unmarshalData[struct {
		EducationStage string `json:"education_stage"`
		List           []struct {
			ID          string `json:"id"`
			Recommended bool   `json:"recommended"`
		}
	}](body)
	if err != nil || catalog.EducationStage != "highschool" {
		t.Fatal(string(body), err)
	}
	for _, s := range catalog.List {
		if s.Recommended && (s.ID == "math_grad" || s.ID == "cs408") {
			t.Fatal("wrong stage recommended", s)
		}
	}
	resp, body, err = doPut(baseURL+"/api/v1/users/me", token, map[string]string{"education_stage": "unknown"})
	if err != nil || resp.StatusCode != 400 {
		t.Fatal(err, string(body))
	}
}
