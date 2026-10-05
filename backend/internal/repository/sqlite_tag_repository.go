package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/kukusuyi/Questrace/backend/internal/domain/model"
)

type SQLiteTagRepository struct {
	db *sql.DB
}

func NewSQLiteTagRepository(db *sql.DB) *SQLiteTagRepository {
	return &SQLiteTagRepository{db: db}
}

func (r *SQLiteTagRepository) List(filter TagFilter) ([]model.Tag, error) {
	query := `
SELECT id, user_id, tag_name, tag_type, usage_count, is_active, deleted_at, created_at, updated_at
FROM tag
WHERE is_active = 1`
	args := make([]any, 0, 3)
	if filter.UserID > 0 {
		query += " AND user_id = ?"
		args = append(args, filter.UserID)
	}
	if strings.TrimSpace(filter.TagType) != "" {
		query += " AND tag_type = ?"
		args = append(args, filter.TagType)
	}
	if strings.TrimSpace(filter.Keyword) != "" {
		query += " AND tag_name LIKE ?"
		args = append(args, "%"+strings.TrimSpace(filter.Keyword)+"%")
	}
	if filter.SubjectID != "" || filter.CourseID != "" {
		query += ` AND EXISTS(SELECT 1 FROM wrong_question_tag qt JOIN wrong_question q ON q.id=qt.question_id WHERE qt.tag_id=tag.id AND q.user_id=tag.user_id AND q.is_deleted=0 AND q.classification_status='confirmed' AND q.analysis_stale=0`
		if filter.SubjectID != "" {
			query += " AND q.subject_id=?"
			args = append(args, filter.SubjectID)
		}
		if filter.CourseID != "" {
			query += " AND q.course_id=?"
			args = append(args, filter.CourseID)
		}
		query += ")"
	}
	query += " ORDER BY usage_count DESC, id DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.Tag, 0)
	for rows.Next() {
		item, scanErr := scanTag(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *SQLiteTagRepository) Ensure(userID int64, tagType, tagName string) (model.Tag, error) {
	_, err := r.db.Exec(
		`INSERT INTO tag (user_id, tag_name, tag_type, usage_count, is_active, deleted_at)
		VALUES (?, ?, ?, 0, 1, NULL)
		ON CONFLICT(user_id, tag_type, tag_name) DO UPDATE SET
			is_active = 1,
			deleted_at = NULL,
			updated_at = CURRENT_TIMESTAMP`,
		userID,
		strings.TrimSpace(tagName),
		tagType,
	)
	if err != nil {
		return model.Tag{}, err
	}

	return r.getByUserTypeName(userID, tagType, tagName)
}

func (r *SQLiteTagRepository) AdjustUsage(userID int64, tagType, tagName string, delta int) error {
	// SQLite triggers update counts in the same transaction as question/tag links.
	return nil
}

func (r *SQLiteTagRepository) Delete(id, userID int64) (model.Tag, bool, error) {
	result, err := r.db.Exec(
		`UPDATE tag
		SET is_active = 0,
		    deleted_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?`,
		id,
		userID,
	)
	if err != nil {
		return model.Tag{}, false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return model.Tag{}, false, err
	}
	if affected == 0 {
		return model.Tag{}, false, nil
	}

	item, err := r.getByID(id)
	if err != nil {
		return model.Tag{}, false, err
	}

	return item, true, nil
}

func (r *SQLiteTagRepository) ResolveNamesByIDs(userID int64, ids []int64) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query := fmt.Sprintf(
		`SELECT tag_name FROM tag WHERE is_active = 1 AND user_id = ? AND id IN (%s) ORDER BY id ASC`,
		buildInt64InClause(ids),
	)
	args := make([]any, 0, len(ids)+1)
	args = append(args, userID)
	for _, id := range ids {
		args = append(args, id)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	names := make([]string, 0, len(ids))
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, scanErr
		}
		names = append(names, name)
	}

	return names, rows.Err()
}

func (r *SQLiteTagRepository) getByID(id int64) (model.Tag, error) {
	row := r.db.QueryRow(
		`SELECT id, user_id, tag_name, tag_type, usage_count, is_active, deleted_at, created_at, updated_at
		FROM tag
		WHERE id = ?`,
		id,
	)
	return scanTag(row)
}

func (r *SQLiteTagRepository) getByUserTypeName(userID int64, tagType, tagName string) (model.Tag, error) {
	row := r.db.QueryRow(
		`SELECT id, user_id, tag_name, tag_type, usage_count, is_active, deleted_at, created_at, updated_at
		FROM tag
		WHERE user_id = ? AND tag_type = ? AND tag_name = ?`,
		userID,
		tagType,
		strings.TrimSpace(tagName),
	)
	item, err := scanTag(row)
	if err != nil {
		return model.Tag{}, err
	}
	return item, nil
}
