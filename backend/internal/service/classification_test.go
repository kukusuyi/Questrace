package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	assets "github.com/kukusuyi/Questrace/backend"
	"github.com/kukusuyi/Questrace/backend/internal/config"
	"github.com/kukusuyi/Questrace/backend/internal/domain/dto"
	"github.com/kukusuyi/Questrace/backend/internal/domain/model"
	"github.com/kukusuyi/Questrace/backend/internal/infra/sqlite"
	"github.com/kukusuyi/Questrace/backend/internal/repository"
)

func classificationDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlite.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err = db.Exec("INSERT INTO user(id,username) VALUES(1,'one'),(2,'two')"); err != nil {
		t.Fatal(err)
	}
	return db
}
func TestCustomSubjectsAndValidation(t *testing.T) {
	s := &SubjectService{DB: classificationDB(t)}
	ctx := SetUserID(context.Background(), 1)
	a, err := s.Create(ctx, "  AP   Physics ")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Create(ctx, "ap physics")
	if err != nil || a.ID != b.ID {
		t.Fatal("duplicate subject", err)
	}
	c, err := s.Create(SetUserID(ctx, 2), "AP Physics")
	if err != nil || c.ID == a.ID {
		t.Fatal("cross-user subject", err)
	}
	if _, err = s.Resolve(ctx, c.ID, "", ""); err == nil {
		t.Fatal("foreign subject accepted")
	}
	for _, bad := range [][3]string{{"highschool_math", "data_structures", ""}, {"cs408", "operating_systems", "排序"}, {"cs408", "", "排序"}, {"math_grad", "", "TCP"}} {
		if _, err = s.Resolve(ctx, bad[0], bad[1], bad[2]); err == nil {
			t.Fatal("invalid combination", bad)
		}
	}
}
func TestSubjectRoutingAndPromptIsolation(t *testing.T) {
	s := &AIService{}
	ctx := SetUserID(context.Background(), 1)
	for _, id := range []string{"math_grad", "cs408", "highschool_math", "highschool_geography", "highschool_biology", "highschool_physics", "highschool_chemistry", ""} {
		t.Run(id, func(t *testing.T) {
			course := ""
			chapter := ""
			if id == "cs408" {
				course = "data_structures"
				chapter = "排序"
			}
			if id == "math_grad" {
				chapter = "矩阵"
			}
			route, _ := json.Marshal(subjectRoute{SubjectID: id, CourseID: course, Chapter: chapter})
			p := &stubAIProvider{responses: []string{string(route), `{"chapter":"","tags":{"knowledge_points":[],"problem_type":[],"method":[],"mistake_reason":["虚构错因"]},"semantic_summary":"摘要","mistake_summary":"不应出现"}`}}
			req, err := s.classify(ctx, p, dto.AnalyzeWrongQuestionRequest{QuestionJSON: dto.QuestionJSON{QuestionCore: "测试题"}})
			if err != nil {
				t.Fatal(err)
			}
			result, err := s.analyzeWithProvider(ctx, p, req)
			if err != nil {
				t.Fatal(err)
			}
			prompt := p.requests[len(p.requests)-1].Messages[0].Content
			if id != "math_grad" && strings.Contains(prompt, "考研线性代数") {
				t.Fatal("math prompt leaked")
			}
			if id == "cs408" && !strings.Contains(prompt, "排序") {
				t.Fatal("408 prompt missing")
			}
			if result.MistakeSummary != "" || len(result.Tags.MistakeReason) != 0 {
				t.Fatal("hallucinated mistake without wrong solution")
			}
		})
	}
	p := &stubAIProvider{responses: []string{`{"subject_id":"math_grad","course_id":"","chapter":"矩阵","conflict":"内容更接近数学"}`}}
	req, err := s.classify(ctx, p, dto.AnalyzeWrongQuestionRequest{SubjectID: "highschool_physics", QuestionJSON: dto.QuestionJSON{QuestionCore: "测试"}})
	if err != nil || req.SubjectID != "highschool_physics" || len(req.Warnings) != 1 {
		t.Fatal("manual selection overwritten", req, err)
	}
}
func Test408PromptsEmbeddedAndLocal(t *testing.T) {
	original := promptFiles
	t.Cleanup(func() { promptFiles = original })
	dir := t.TempDir()
	err := fs.WalkDir(assets.Files, "prompts", func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() {
			return os.MkdirAll(filepath.Join(dir, path), 0700)
		}
		b, err := fs.ReadFile(assets.Files, path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, path), b, 0600)
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []fs.FS{assets.Files, os.DirFS(dir)} {
		promptFiles = source
		items, err := builtinSubjects()
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, course := range items[1].Courses {
			prompts, err := scopedPrompts("cs408", course.ID)
			if err != nil || len(prompts) != len(course.Chapters) {
				t.Fatal(course.ID, err)
			}
			for _, p := range prompts {
				count++
				for _, key := range []string{"knowledge_points", "problem_type", "method", "mistake_reason", "易错边界"} {
					if !strings.Contains(p.Content, key) {
						t.Fatal(p.Name, key)
					}
				}
			}
		}
		if count != 26 {
			t.Fatal("missing chapters", count)
		}
	}
}
func TestIdenticalVectorsNeverCrossClassification(t *testing.T) {
	db := classificationDB(t)
	repo := repository.NewSQLiteQuestionRepository(db)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"embedding": []float64{1, 0, 0}}}})
	}))
	defer server.Close()
	cfg := config.EmbeddingModelConfig{ProviderType: "openai_compatible", BaseURL: server.URL, Model: "test", APIKey: "test"}
	local := &LocalVector{DB: db, Config: func() config.EmbeddingModelConfig { return cfg }}
	makeQ := func(uid int64, subject, course, status string) model.WrongQuestion {
		t.Helper()
		q, err := repo.Create(model.WrongQuestion{UserID: uid, Subject: subject, SubjectID: subject, CourseID: course, ClassificationStatus: status, QuestionCore: "identical", MasteryStatus: "unmastered", SourceType: "manual", CreatedAt: time.Now(), UpdatedAt: time.Now()})
		if err != nil {
			t.Fatal(err)
		}
		return q
	}
	base := makeQ(1, "cs408", "data_structures", "confirmed")
	same := makeQ(1, "cs408", "data_structures", "confirmed")
	otherCourse := makeQ(1, "cs408", "operating_systems", "confirmed")
	makeQ(1, "math_grad", "", "confirmed")
	makeQ(1, "highschool_math", "", "confirmed")
	makeQ(2, "cs408", "data_structures", "confirmed")
	makeQ(1, "cs408", "data_structures", "legacy_pending")
	for i := 0; i < 7; i++ {
		local.step(context.Background())
	}
	for _, tags := range []bool{false, true} {
		got, err := local.Search(base, "semantic", 20, tags)
		if err != nil || len(got) != 1 || got[0].QuestionID != same.ID {
			t.Fatal("cross classification", got, err)
		}
	}
	base.RecallScope = "subject"
	got, err := local.Search(base, "semantic", 20, false)
	if err != nil || len(got) != 2 {
		t.Fatal("408 scope", got, err)
	}
	for _, x := range got {
		if x.QuestionID != same.ID && x.QuestionID != otherCourse.ID {
			t.Fatal("foreign result", got)
		}
	}
	base.ClassificationStatus = "pending"
	if _, err = local.Search(base, "semantic", 20, false); err == nil {
		t.Fatal("pending query accepted")
	}
	base.ClassificationStatus = "confirmed"
	base.SubjectID = ""
	if _, err = local.Search(base, "semantic", 20, false); err == nil {
		t.Fatal("missing subject accepted")
	}
	var mistakes int
	if err = db.QueryRow("SELECT count(*) FROM local_vector WHERE vector_type='mistake'").Scan(&mistakes); err != nil || mistakes != 0 {
		t.Fatal("empty mistake vectors", mistakes, err)
	}
	// Format compatibility is independent of matching dimensions/model.
	if _, err = db.Exec("UPDATE local_vector SET format_version=1 WHERE question_id=?", same.ID); err != nil {
		t.Fatal(err)
	}
	base.SubjectID = "cs408"
	base.RecallScope = "course"
	got, err = local.Search(base, "semantic", 20, false)
	if err != nil || len(got) != 0 {
		t.Fatal("old format returned", got, err)
	}
	// Both API entry points share the same boundaries, including missing JSON classification.
	svc := NewQuestionService(repo, nil, nil, &VectorService{Local: local})
	svc.Subjects = &SubjectService{DB: db}
	ctx := SetUserID(context.Background(), 1)
	if _, err = svc.SimilarByJSON(ctx, dto.SimilarByJSONRequest{QuestionJSON: dto.QuestionJSON{QuestionCore: "q"}}); err == nil {
		t.Fatal("unclassified JSON searched globally")
	}
	for _, status := range []string{"pending", "legacy_pending"} {
		if _, err = svc.SimilarByJSON(ctx, dto.SimilarByJSONRequest{SubjectID: "cs408", CourseID: "operating_systems", ClassificationStatus: status, QuestionJSON: dto.QuestionJSON{QuestionCore: "q"}}); err == nil {
			t.Fatal("unconfirmed JSON was recalled", status)
		}
	}
	out, err := svc.SimilarByJSON(ctx, dto.SimilarByJSONRequest{SubjectID: "cs408", CourseID: "operating_systems", QuestionJSON: dto.QuestionJSON{QuestionCore: "q"}})
	if err != nil || len(out.List) != 1 || out.List[0].QuestionID != otherCourse.ID {
		t.Fatal(out, err)
	}
}
func TestClassificationBatchIsAtomicAndInvalidatesVectors(t *testing.T) {
	db := classificationDB(t)
	s := &SubjectService{DB: db}
	ctx := SetUserID(context.Background(), 1)
	_, err := db.Exec(`INSERT INTO wrong_question(id,user_id,subject,chapter,question_core,semantic_summary,mistake_summary,classification_status) VALUES(1,1,'math','原章节','原题','原摘要','原错因','legacy_pending'),(2,2,'math','','私有题','','','legacy_pending'); INSERT INTO local_vector(question_id,vector_type,model,dimension,content_hash,vector,format_version) VALUES(1,'semantic','test',1,'test',x'00000000',2)`)
	if err != nil {
		t.Fatal(err)
	}
	in := ClassificationBatch{SubjectID: "highschool_physics", Items: []ClassificationTarget{{1, 1}, {2, 1}}}
	if err = s.Reclassify(ctx, in); err == nil {
		t.Fatal("foreign batch accepted")
	}
	var subject string
	db.QueryRow("SELECT subject FROM wrong_question WHERE id=1").Scan(&subject)
	if subject != "math" {
		t.Fatal("partial mutation")
	}
	in.Items = in.Items[:1]
	if err = s.Reclassify(ctx, in); err != nil {
		t.Fatal(err)
	}
	q, ok := repository.NewSQLiteQuestionRepository(db).GetByID(1)
	if !ok || q.Revision != 2 || q.SubjectID != "highschool_physics" || q.Chapter != "" || q.SemanticSummary != "原摘要" || !q.AnalysisStale {
		t.Fatal(q)
	}
	var n int
	db.QueryRow("SELECT count(*) FROM local_vector WHERE question_id=1").Scan(&n)
	if n != 0 {
		t.Fatal("stale vector survived")
	}
	if err = s.Reclassify(ctx, in); err == nil {
		t.Fatal("stale revision accepted")
	}
	if strings.Contains(buildSearchText(q, "semantic"), "原摘要") {
		t.Fatal("stale summary indexed")
	}
}
func TestIndexJobCannotCommitAfterClassificationChange(t *testing.T) {
	db := classificationDB(t)
	repo := repository.NewSQLiteQuestionRepository(db)
	q, err := repo.Create(model.WrongQuestion{UserID: 1, Subject: "数学", SubjectID: "math_grad", ClassificationStatus: "confirmed", QuestionCore: "q", SourceType: "manual", MasteryStatus: "unmastered", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	once := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if once {
			once = false
			if _, e := db.Exec("UPDATE wrong_question SET subject_id='highschool_physics' WHERE id=?", q.ID); e != nil {
				t.Error(e)
			}
		}
		json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"embedding": []float64{1}}}})
	}))
	defer server.Close()
	local := &LocalVector{DB: db, Config: func() config.EmbeddingModelConfig {
		return config.EmbeddingModelConfig{ProviderType: "openai_compatible", BaseURL: server.URL, Model: "test", APIKey: "test"}
	}}
	local.step(context.Background())
	var n int
	db.QueryRow("SELECT count(*) FROM local_vector").Scan(&n)
	if n != 0 {
		t.Fatal("stale job committed")
	}
	local.step(context.Background())
	db.QueryRow("SELECT count(*) FROM local_vector").Scan(&n)
	if n != 1 {
		t.Fatal("new revision not indexed")
	}
}

func TestSpecializedTagsExcludeInstructionText(t *testing.T) {
	result := llmAnalyzeResult{Tags: dto.TagGroups{KnowledgePoints: []string{"快速排序", "数学函数"}, MistakeReason: []string{"边界错误", "不是标签"}}}
	constrainTags(&result, "## knowledge_points\n### 子类\n- 快速排序\n## mistake_reason\n- 边界错误\n## 易错边界\n- 不是标签\n")
	if len(result.Tags.KnowledgePoints) != 1 || result.Tags.KnowledgePoints[0] != "快速排序" || len(result.Tags.MistakeReason) != 1 || result.Tags.MistakeReason[0] != "边界错误" {
		t.Fatalf("instructions leaked into tag vocabulary: %+v", result.Tags)
	}
}

func TestStageProfilesAndOpenEndedSubjects(t *testing.T) {
	db := classificationDB(t)
	subjects := &SubjectService{DB: db}
	ctx := SetUserID(context.Background(), 1)
	users := NewUserService(repository.NewSQLiteUserRepository(db))
	me, err := users.GetMe(ctx)
	if err != nil || me.EducationStage != "university" {
		t.Fatal(me, err)
	}
	materials, err := subjects.Create(ctx, "材料科学基础")
	if err != nil {
		t.Fatal(err)
	}
	mechanical, err := subjects.Create(ctx, "机械原理")
	if err != nil {
		t.Fatal(err)
	}
	all, err := subjects.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	university := SubjectsForStage(all, "university", "")
	for _, s := range university {
		if strings.HasPrefix(s.ID, "highschool_") {
			t.Fatal("highschool branch exposed", s)
		}
	}
	if len(university) != 4 {
		t.Fatal("professional subjects missing", university)
	}
	provider := &stubAIProvider{responses: []string{`{"subject_id":"","suggested_subject":"临床医学"}`}}
	aiService := &AIService{Subjects: subjects}
	routed, err := aiService.classify(ctx, provider, dto.AnalyzeWrongQuestionRequest{QuestionJSON: dto.QuestionJSON{QuestionCore: "医学测试题"}})
	if err != nil || routed.SubjectID != "" || routed.SuggestedSubject != "临床医学" {
		t.Fatal(routed, err)
	}
	payload := provider.requests[0].Messages[1].Content
	if !strings.Contains(payload, materials.ID) || !strings.Contains(payload, mechanical.ID) || strings.Contains(payload, "highschool_math") {
		t.Fatal("incorrect university catalog", payload)
	}
	if _, err = users.UpdateEducationStage(ctx, "highschool"); err != nil {
		t.Fatal(err)
	}
	me, err = users.GetMe(ctx)
	if err != nil || me.EducationStage != "highschool" {
		t.Fatal(me, err)
	}
	other, err := users.GetMe(SetUserID(ctx, 2))
	if err != nil || other.EducationStage != "university" {
		t.Fatal("another account changed", other, err)
	}
	if _, err = users.UpdateEducationStage(ctx, "primary"); err == nil {
		t.Fatal("invalid stage accepted")
	}
	provider = &stubAIProvider{responses: []string{`{"subject_id":"highschool_math"}`}}
	if _, err = aiService.classify(ctx, provider, dto.AnalyzeWrongQuestionRequest{QuestionJSON: dto.QuestionJSON{QuestionCore: "三角函数基础题"}}); err != nil {
		t.Fatal(err)
	}
	payload = provider.requests[0].Messages[1].Content
	if strings.Contains(payload, "math_grad") || !strings.Contains(payload, "highschool_math") {
		t.Fatal("stage ignored", payload)
	}
	// Changing a preference must not prevent reading or explicitly editing an existing subject.
	if _, err = subjects.Resolve(ctx, materials.ID, "", "晶体结构"); err != nil {
		t.Fatal(err)
	}
}
