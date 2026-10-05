package model

import "time"

type WrongQuestion struct {
	SubjectID            string
	CourseID             string
	ClassificationStatus string
	AnalysisStale        bool
	Revision             int64
	RecallScope          string // query-only: course (default) or subject

	ID               int64
	UserID           int64
	Subject          string
	Chapter          string
	QuestionCore     string
	StandardSolution string
	WrongSolution    string
	SemanticSummary  string
	MistakeSummary   string
	DifficultyLevel  int
	MasteryStatus    string
	SourceType       string
	SourceImageID    *int64
	SourceImageURL   string
	IsDeleted        bool
	DeletedAt        *time.Time
	Tags             TagCollections
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
