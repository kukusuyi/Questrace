package repository

import (
	"database/sql"

	"github.com/kukusuyi/Questrace/backend/internal/domain/model"
)

type UserRepository interface {
	UpdateEducationStage(id int64, stage string) error
	Create(user model.User) (model.User, error)
	GetByUsername(username string) (model.User, bool, error)
	GetByEmail(email string) (model.User, bool, error)
	GetByID(id int64) (model.User, bool, error)
}

type SQLiteUserRepository struct {
	db *sql.DB
}

func NewSQLiteUserRepository(db *sql.DB) *SQLiteUserRepository {
	return &SQLiteUserRepository{db: db}
}

func (r *SQLiteUserRepository) Create(user model.User) (model.User, error) {
	if user.EducationStage == "" {
		user.EducationStage = "university"
	}
	result, err := r.db.Exec(
		"INSERT INTO `user` (username, email, password_hash, role, education_stage) VALUES (?, ?, ?, ?, ?)",
		user.Username, user.Email, user.PasswordHash, user.Role, user.EducationStage,
	)
	if err != nil {
		return model.User{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return model.User{}, err
	}

	user.ID = id
	return user, nil
}

func (r *SQLiteUserRepository) GetByUsername(username string) (model.User, bool, error) {
	return r.getOne("SELECT id, username, coalesce(email,''), coalesce(password_hash,''), role, created_at, updated_at, education_stage FROM `user` WHERE username = ?", username)
}

func (r *SQLiteUserRepository) GetByEmail(email string) (model.User, bool, error) {
	return r.getOne("SELECT id, username, coalesce(email,''), coalesce(password_hash,''), role, created_at, updated_at, education_stage FROM `user` WHERE email = ?", email)
}

func (r *SQLiteUserRepository) GetByID(id int64) (model.User, bool, error) {
	return r.getOne("SELECT id, username, coalesce(email,''), coalesce(password_hash,''), role, created_at, updated_at, education_stage FROM `user` WHERE id = ?", id)
}

func (r *SQLiteUserRepository) getOne(query string, arg any) (model.User, bool, error) {
	row := r.db.QueryRow(query, arg)

	var user model.User
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt, &user.EducationStage)
	if err == sql.ErrNoRows {
		return model.User{}, false, nil
	}
	if err != nil {
		return model.User{}, false, err
	}

	return user, true, nil
}

func (r *SQLiteUserRepository) UpdateEducationStage(id int64, stage string) error {
	_, err := r.db.Exec("UPDATE user SET education_stage=?,updated_at=CURRENT_TIMESTAMP WHERE id=?", stage, id)
	return err
}
