package dto

type QuestionJSON struct {
	QuestionCore     string `json:"question_core"`
	StandardSolution string `json:"standard_solution"`
	WrongSolution    string `json:"wrong_solution"`
}

type TagGroups struct {
	KnowledgePoints []string `json:"knowledge_points"`
	ProblemType     []string `json:"problem_type"`
	Method          []string `json:"method"`
	MistakeReason   []string `json:"mistake_reason"`
}

type CreateWrongQuestionRequest struct {
	AnalysisConfirmed    bool   `json:"analysis_confirmed"`
	SubjectID            string `json:"subject_id"`
	CourseID             string `json:"course_id"`
	ClassificationStatus string `json:"classification_status"`
	AnalysisStale        bool   `json:"analysis_stale"`
	Revision             int64  `json:"revision"`

	SourceType      string       `json:"source_type"`
	SourceImageID   *int64       `json:"source_image_id"`
	SourceImageURL  string       `json:"source_image_url"`
	Subject         string       `json:"subject"`
	Chapter         string       `json:"chapter"`
	QuestionJSON    QuestionJSON `json:"question_json"`
	Tags            TagGroups    `json:"tags"`
	SemanticSummary string       `json:"semantic_summary"`
	MistakeSummary  string       `json:"mistake_summary"`
	DifficultyLevel int          `json:"difficulty_level"`
	MasteryStatus   string       `json:"mastery_status"`
}

type UpdateWrongQuestionRequest struct {
	AnalysisConfirmed    bool   `json:"analysis_confirmed"`
	SubjectID            string `json:"subject_id"`
	CourseID             string `json:"course_id"`
	ClassificationStatus string `json:"classification_status"`
	AnalysisStale        bool   `json:"analysis_stale"`
	Revision             int64  `json:"revision"`

	QuestionJSON    QuestionJSON `json:"question_json"`
	Subject         string       `json:"subject"`
	Chapter         string       `json:"chapter"`
	Tags            TagGroups    `json:"tags"`
	SourceImageID   *int64       `json:"source_image_id"`
	SourceImageURL  string       `json:"source_image_url"`
	SemanticSummary string       `json:"semantic_summary"`
	MistakeSummary  string       `json:"mistake_summary"`
	DifficultyLevel int          `json:"difficulty_level"`
	MasteryStatus   string       `json:"mastery_status"`
}

type QuestionListItem struct {
	SubjectID            string `json:"subject_id"`
	CourseID             string `json:"course_id"`
	ClassificationStatus string `json:"classification_status"`
	AnalysisStale        bool   `json:"analysis_stale"`
	Revision             int64  `json:"revision"`

	QuestionID      int64     `json:"question_id"`
	QuestionCore    string    `json:"question_core"`
	SourceImageID   *int64    `json:"source_image_id"`
	SourceImageURL  string    `json:"source_image_url"`
	Subject         string    `json:"subject"`
	Chapter         string    `json:"chapter"`
	Tags            TagGroups `json:"tags"`
	DifficultyLevel int       `json:"difficulty_level"`
	MasteryStatus   string    `json:"mastery_status"`
	CreatedAt       string    `json:"created_at"`
}

type QuestionDetail struct {
	SubjectID            string `json:"subject_id"`
	CourseID             string `json:"course_id"`
	ClassificationStatus string `json:"classification_status"`
	AnalysisStale        bool   `json:"analysis_stale"`
	Revision             int64  `json:"revision"`

	QuestionID       int64     `json:"question_id"`
	QuestionCore     string    `json:"question_core"`
	StandardSolution string    `json:"standard_solution"`
	WrongSolution    string    `json:"wrong_solution"`
	SemanticSummary  string    `json:"semantic_summary"`
	MistakeSummary   string    `json:"mistake_summary"`
	SourceType       string    `json:"source_type"`
	SourceImageID    *int64    `json:"source_image_id"`
	SourceImageURL   string    `json:"source_image_url"`
	Subject          string    `json:"subject"`
	Chapter          string    `json:"chapter"`
	Tags             TagGroups `json:"tags"`
	DifficultyLevel  int       `json:"difficulty_level"`
	MasteryStatus    string    `json:"mastery_status"`
	CreatedAt        string    `json:"created_at"`
	UpdatedAt        string    `json:"updated_at"`
}

type QuestionExportItem struct {
	SubjectID            string `json:"subject_id"`
	CourseID             string `json:"course_id"`
	ClassificationStatus string `json:"classification_status"`
	AnalysisStale        bool   `json:"analysis_stale"`
	Revision             int64  `json:"revision"`

	QuestionID       int64
	QuestionCore     string
	StandardSolution string
	WrongSolution    string
	SemanticSummary  string
	MistakeSummary   string
	SourceType       string
	SourceImageURL   string
	Subject          string
	Chapter          string
	Tags             TagGroups
	DifficultyLevel  int
	MasteryStatus    string
	CreatedAt        string
	UpdatedAt        string
}

type CreateWrongQuestionResponse struct {
	QuestionID int64 `json:"question_id"`
}

type UpdateWrongQuestionResponse struct {
	QuestionID int64 `json:"question_id"`
	Updated    bool  `json:"updated"`
}

type DeleteWrongQuestionResponse struct {
	QuestionID int64 `json:"question_id"`
	Deleted    bool  `json:"deleted"`
}

type SimilarQuestionRequest struct {
	RecallScope  string `json:"recall_scope"`
	VectorType   string `json:"vector_type"`
	Limit        int    `json:"limit"`
	UseTagFilter bool   `json:"use_tag_filter"`
}

type SimilarByJSONRequest struct {
	Chapter              string `json:"chapter"`
	SubjectID            string `json:"subject_id"`
	CourseID             string `json:"course_id"`
	ClassificationStatus string `json:"classification_status"`
	AnalysisStale        bool   `json:"analysis_stale"`
	Revision             int64  `json:"revision"`
	RecallScope          string `json:"recall_scope"`

	QuestionJSON QuestionJSON `json:"question_json"`
	Tags         TagGroups    `json:"tags"`
	VectorType   string       `json:"vector_type"`
	Limit        int          `json:"limit"`
	UseTagFilter bool         `json:"use_tag_filter"`
}

type SimilarQuestionItem struct {
	Subject              string `json:"subject"`
	Chapter              string `json:"chapter"`
	SubjectID            string `json:"subject_id"`
	CourseID             string `json:"course_id"`
	ClassificationStatus string `json:"classification_status"`
	AnalysisStale        bool   `json:"analysis_stale"`
	Revision             int64  `json:"revision"`

	QuestionID     int64     `json:"question_id"`
	Score          float64   `json:"score"`
	SimilarityType string    `json:"similarity_type"`
	QuestionCore   string    `json:"question_core"`
	SourceImageID  *int64    `json:"source_image_id"`
	SourceImageURL string    `json:"source_image_url"`
	MatchedTags    []string  `json:"matched_tags"`
	Reason         string    `json:"reason"`
	Tags           TagGroups `json:"tags"`
}

type SimilarQuestionResponse struct {
	Status string                `json:"status"`
	List   []SimilarQuestionItem `json:"list"`
}

type ListQuestionFilter struct {
	SubjectID            string
	CourseID             string
	ClassificationStatus string
	Page                 int
	PageSize             int
	Subject              string
	Chapter              string
	Keyword              string
	TagIDs               []int64
	MasteryStatus        string
	DifficultyLevel      int
	SourceType           string
}

type CreateTagRequest struct {
	TagName string `json:"tag_name"`
	TagType string `json:"tag_type"`
}

type TagItem struct {
	TagID      int64  `json:"tag_id"`
	TagName    string `json:"tag_name"`
	TagType    string `json:"tag_type"`
	UsageCount int    `json:"usage_count"`
	IsActive   bool   `json:"is_active"`
}

type TagListResponse struct {
	List []TagItem `json:"list"`
}

type DeleteTagResponse struct {
	TagID   int64 `json:"tag_id"`
	Deleted bool  `json:"deleted"`
}

type FileUploadResponse struct {
	ImageID  int64  `json:"image_id"`
	ImageURL string `json:"image_url"`
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
	MIMEType string `json:"mime_type"`
}

type UserMeResponse struct {
	EducationStage string `json:"education_stage"`
	UserID         int64  `json:"user_id"`
	Username       string `json:"username"`
	Email          string `json:"email"`
	CreatedAt      string `json:"created_at"`
}
