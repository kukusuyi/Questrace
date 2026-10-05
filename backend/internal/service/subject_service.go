package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/kukusuyi/Questrace/backend/internal/domain/model"
	apperrors "github.com/kukusuyi/Questrace/backend/internal/pkg/errors"
)

type SubjectCourse struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Chapters []string `json:"chapters"`
}
type SubjectItem struct {
	Recommended    bool            `json:"recommended"`
	EducationStage string          `json:"education_stage"`
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Specialized    bool            `json:"specialized"`
	Courses        []SubjectCourse `json:"courses"`
	Chapters       []string        `json:"chapters"`
}
type SubjectService struct{ DB *sql.DB }

func builtinSubjects() ([]SubjectItem, error) {
	math, err := discoverAnalyzeChapterPrompts()
	if err != nil {
		return nil, err
	}
	items := []SubjectItem{
		{ID: "math_grad", Name: "考研数学", Specialized: true, Chapters: chapterPromptNames(math), Courses: []SubjectCourse{}},
		{ID: "cs408", Name: "408", Specialized: true, Chapters: []string{}, Courses: []SubjectCourse{
			{ID: "data_structures", Name: "数据结构", Chapters: strings.Split("算法基础与复杂度、线性表、栈队列与数组、串、树与二叉树、图、查找、排序", "、")},
			{ID: "computer_organization", Name: "计算机组成原理", Chapters: strings.Split("计算机系统概述、数据表示与运算、存储系统、指令系统、中央处理器、总线、输入输出系统", "、")},
			{ID: "operating_systems", Name: "操作系统", Chapters: strings.Split("操作系统概述、进程与线程、内存管理、文件管理、输入输出管理", "、")},
			{ID: "computer_networks", Name: "计算机网络", Chapters: strings.Split("网络体系结构、物理层、数据链路层、网络层、传输层、应用层", "、")},
		}},
		{ID: "highschool_math", Name: "高中数学", Chapters: []string{}, Courses: []SubjectCourse{}},
		{ID: "highschool_geography", Name: "高中地理", Chapters: []string{}, Courses: []SubjectCourse{}},
		{ID: "highschool_biology", Name: "高中生物", Chapters: []string{}, Courses: []SubjectCourse{}},
		{ID: "highschool_physics", Name: "高中物理", Chapters: []string{}, Courses: []SubjectCourse{}},
		{ID: "highschool_chemistry", Name: "高中化学", Chapters: []string{}, Courses: []SubjectCourse{}},
	}
	for i := range items {
		items[i].EducationStage = "university"
		if strings.HasPrefix(items[i].ID, "highschool_") {
			items[i].EducationStage = "highschool"
		}
	}
	return items, nil
}
func (s *SubjectService) List(ctx context.Context) ([]SubjectItem, error) {
	uid, err := RequireUserID(ctx)
	if err != nil {
		return nil, err
	}
	items, err := builtinSubjects()
	if err != nil {
		return nil, err
	}
	if s == nil || s.DB == nil {
		return items, nil
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT id,name,education_stage FROM custom_subject WHERE user_id=? ORDER BY name,id", uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		x := SubjectItem{Courses: []SubjectCourse{}, Chapters: []string{}}
		if err = rows.Scan(&x.ID, &x.Name, &x.EducationStage); err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
func (s *SubjectService) Create(ctx context.Context, name string) (SubjectItem, error) {
	uid, err := RequireUserID(ctx)
	if err != nil {
		return SubjectItem{}, err
	}
	name = strings.Join(strings.Fields(name), " ")
	if name == "" || utf8.RuneCountInString(name) > 64 {
		return SubjectItem{}, apperrors.New(400, 40001, "学科名称应为1至64个字符")
	}
	normalized := strings.ToLower(name)
	items, err := s.List(ctx)
	if err != nil {
		return SubjectItem{}, err
	}
	for _, x := range items {
		if strings.ToLower(x.Name) == normalized {
			return x, nil
		}
	}
	if s == nil || s.DB == nil {
		return SubjectItem{}, apperrors.New(503, 50300, "学科目录暂不可用")
	}
	stage, err := s.EducationStage(ctx)
	if err != nil {
		return SubjectItem{}, err
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", uid, normalized)))
	id := "custom_" + hex.EncodeToString(sum[:16])
	_, err = s.DB.ExecContext(ctx, "INSERT INTO custom_subject(id,user_id,name,normalized_name,education_stage) VALUES(?,?,?,?,?) ON CONFLICT(user_id,normalized_name) DO NOTHING", id, uid, name, normalized, stage)
	if err != nil {
		return SubjectItem{}, err
	}
	var x SubjectItem
	x.Courses = []SubjectCourse{}
	x.Chapters = []string{}
	err = s.DB.QueryRowContext(ctx, "SELECT id,name,education_stage FROM custom_subject WHERE user_id=? AND normalized_name=?", uid, normalized).Scan(&x.ID, &x.Name, &x.EducationStage)
	return x, err
}
func (s *SubjectService) Resolve(ctx context.Context, id, course, chapter string) (SubjectItem, error) {
	items, err := s.List(ctx)
	if err != nil {
		return SubjectItem{}, err
	}
	for _, x := range items {
		if x.ID != id {
			continue
		}
		chapters := x.Chapters
		if len(x.Courses) > 0 {
			if course == "" {
				if chapter != "" {
					return SubjectItem{}, apperrors.New(400, 40001, "请先选择408课程")
				}
				return x, nil
			}
			found := false
			for _, c := range x.Courses {
				if c.ID == course {
					found = true
					chapters = c.Chapters
					break
				}
			}
			if !found {
				return SubjectItem{}, apperrors.New(400, 40001, "课程不属于所选学科")
			}
		} else if course != "" {
			return SubjectItem{}, apperrors.New(400, 40001, "该学科没有课程分组")
		}
		if x.Specialized && chapter != "" && !slices.Contains(chapters, chapter) {
			return SubjectItem{}, apperrors.New(400, 40001, "章节不属于所选学科或课程")
		}
		return x, nil
	}
	return SubjectItem{}, apperrors.New(400, 40001, "学科不存在，请重新选择")
}
func (s *SubjectService) Apply(ctx context.Context, q *model.WrongQuestion, id, course, chapter string) error {
	id = strings.TrimSpace(id)
	course = strings.TrimSpace(course)
	chapter = strings.TrimSpace(chapter)
	if id == "" {
		if course != "" {
			return apperrors.New(400, 40001, "请先选择学科")
		}
		q.SubjectID = ""
		q.CourseID = ""
		q.ClassificationStatus = "pending"
		q.Chapter = chapter
		if q.Subject == "" {
			q.Subject = "待分类"
		}
		return nil
	}
	x, err := s.Resolve(ctx, id, course, chapter)
	if err != nil {
		return err
	}
	q.SubjectID = id
	q.Subject = x.Name
	q.CourseID = course
	q.Chapter = chapter
	q.ClassificationStatus = "confirmed"
	return nil
}

type ClassificationTarget struct {
	QuestionID int64 `json:"question_id"`
	Revision   int64 `json:"revision"`
}
type ClassificationBatch struct {
	Items     []ClassificationTarget `json:"items"`
	SubjectID string                 `json:"subject_id"`
	CourseID  string                 `json:"course_id"`
	Chapter   string                 `json:"chapter"`
}

func (s *SubjectService) Reclassify(ctx context.Context, in ClassificationBatch) error {
	uid, err := RequireUserID(ctx)
	if err != nil {
		return err
	}
	if len(in.Items) == 0 || len(in.Items) > 100 {
		return apperrors.New(400, 40001, "每次请选择1至100道错题")
	}
	if in.SubjectID == "" {
		return apperrors.New(400, 40001, "请选择目标学科")
	}
	target, err := s.Resolve(ctx, in.SubjectID, in.CourseID, in.Chapter)
	if err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	seen := map[int64]bool{}
	for _, item := range in.Items {
		if seen[item.QuestionID] || item.Revision < 1 {
			return apperrors.New(400, 40001, "题目重复或缺少版本，请刷新列表")
		}
		seen[item.QuestionID] = true
		var oldID, oldCourse, oldChapter string
		if err = tx.QueryRowContext(ctx, "SELECT subject_id,course_id,coalesce(chapter,'') FROM wrong_question WHERE id=? AND user_id=? AND is_deleted=0 AND revision=?", item.QuestionID, uid, item.Revision).Scan(&oldID, &oldCourse, &oldChapter); err == sql.ErrNoRows {
			return apperrors.New(409, 40901, "题目已变更或不可访问，请刷新后重新选择")
		}
		if err != nil {
			return err
		}
		chapter := in.Chapter
		if chapter == "" && oldID == in.SubjectID && oldCourse == in.CourseID {
			chapter = oldChapter
		}
		_, err = tx.ExecContext(ctx, `UPDATE wrong_question SET subject_id=?,subject=?,course_id=?,chapter=?,classification_status='confirmed',analysis_stale=CASE WHEN subject_id<>? OR course_id<>? OR coalesce(chapter,'')<>? THEN 1 ELSE analysis_stale END,updated_at=CURRENT_TIMESTAMP WHERE id=? AND user_id=? AND revision=?`, in.SubjectID, target.Name, in.CourseID, chapter, in.SubjectID, in.CourseID, chapter, item.QuestionID, uid, item.Revision)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Stage controls recommendations and automatic recognition, never reclassifies saved questions.
func (s *SubjectService) EducationStage(ctx context.Context) (string, error) {
	uid, err := RequireUserID(ctx)
	if err != nil {
		return "", err
	}
	if s == nil || s.DB == nil {
		return "university", nil
	}
	var stage string
	err = s.DB.QueryRowContext(ctx, "SELECT education_stage FROM user WHERE id=?", uid).Scan(&stage)
	return stage, err
}
func SubjectsForStage(items []SubjectItem, stage, selectedID string) []SubjectItem {
	out := []SubjectItem{}
	for _, s := range items {
		if s.EducationStage == stage || s.ID == selectedID {
			out = append(out, s)
		}
	}
	return out
}
