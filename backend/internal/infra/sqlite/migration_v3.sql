CREATE TABLE custom_subject (id TEXT PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES user(id), name TEXT NOT NULL, normalized_name TEXT NOT NULL, UNIQUE(user_id, normalized_name));
ALTER TABLE wrong_question ADD COLUMN subject_id TEXT NOT NULL DEFAULT '';
ALTER TABLE wrong_question ADD COLUMN course_id TEXT NOT NULL DEFAULT '';
ALTER TABLE wrong_question ADD COLUMN classification_status TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE wrong_question ADD COLUMN analysis_stale INTEGER NOT NULL DEFAULT 0;
ALTER TABLE wrong_question ADD COLUMN revision INTEGER NOT NULL DEFAULT 1;
UPDATE wrong_question SET classification_status='legacy_pending', analysis_stale=1;
ALTER TABLE local_vector ADD COLUMN format_version INTEGER NOT NULL DEFAULT 1;
DELETE FROM local_vector;
UPDATE vector_job SET status='done', revision=revision+1;
DROP TRIGGER queue_question_update;
CREATE TRIGGER queue_question_update AFTER UPDATE OF question_core,standard_solution,wrong_solution,semantic_summary,mistake_summary,subject,chapter,subject_id,course_id,classification_status,analysis_stale,is_deleted ON wrong_question BEGIN
 UPDATE wrong_question SET revision=OLD.revision+1 WHERE id=NEW.id;
 INSERT INTO vector_job(question_id) VALUES(NEW.id) ON CONFLICT(question_id) DO UPDATE SET revision=revision+1,status='pending',attempts=0,next_attempt=0;
 DELETE FROM local_vector WHERE question_id=NEW.id;
END;
CREATE INDEX idx_question_classification ON wrong_question(user_id, subject_id, course_id, classification_status, is_deleted);
