package dto

type AnalyzeWrongQuestionRequest struct {
	SubjectID        string   `json:"subject_id"`
	Subject          string   `json:"subject"`
	CourseID         string   `json:"course_id"`
	SuggestedSubject string   `json:"suggested_subject"`
	Warnings         []string `json:"warnings"`

	ProviderName string       `json:"provider_name"`
	ModelName    string       `json:"model_name"`
	Chapter      string       `json:"chapter"`
	QuestionJSON QuestionJSON `json:"question_json"`
	OCRContext   *OCRContext  `json:"ocr_context,omitempty"`
}

type AnalyzeWrongQuestionResponse struct {
	SubjectID            string   `json:"subject_id"`
	Subject              string   `json:"subject"`
	CourseID             string   `json:"course_id"`
	SuggestedSubject     string   `json:"suggested_subject"`
	Warnings             []string `json:"warnings"`
	ClassificationStatus string   `json:"classification_status"`

	Chapter         string    `json:"chapter"`
	Tags            TagGroups `json:"tags"`
	SemanticSummary string    `json:"semantic_summary"`
	MistakeSummary  string    `json:"mistake_summary"`
}

type AIProviderItem struct {
	ProviderName    string `json:"provider_name"`
	ProviderType    string `json:"provider_type"`
	ConfiguredModel string `json:"configured_model"`
}

type AIProviderListResponse struct {
	List []AIProviderItem `json:"list"`
}

type AIProviderModelItem struct {
	ModelName string `json:"model_name"`
}

type AIProviderModelListResponse struct {
	ProviderName string                `json:"provider_name"`
	List         []AIProviderModelItem `json:"list"`
}

type AIChapterListResponse struct {
	List []string `json:"list"`
}
