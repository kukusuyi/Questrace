package dto

type RegisterRequest struct {
	EducationStage string `json:"education_stage"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	Email          string `json:"email"`
}

type RegisterResponse struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Token    string `json:"token"`
}
