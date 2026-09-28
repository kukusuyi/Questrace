package model

import "time"

type User struct {
	EducationStage string
	ID             int64
	Username       string
	Email          string
	PasswordHash   string
	Role           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
