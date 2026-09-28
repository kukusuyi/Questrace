package sqlite

import (
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDatabaseURL(t *testing.T) {
	for _, path := range []string{"C:/Users/中文 data/questrace.db", "/tmp/中文 data/questrace.db"} {
		uri := databaseURL(path)
		parsed, err := url.Parse(uri.String())
		if err != nil || parsed.Host != "" || parsed.Path[0] != '/' {
			t.Fatalf("invalid SQLite URI: %s", uri.String())
		}
	}
}

func TestResolvePath(t *testing.T) {
	touch := func(t *testing.T, path string) {
		t.Helper()
		if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	// A fresh directory gets the new file name.
	dir := t.TempDir()
	path, err := ResolvePath(dir)
	if err != nil || path != filepath.Join(dir, DatabaseName) {
		t.Fatalf("fresh install: %q %v", path, err)
	}

	// A pre-rename database keeps being used in place.
	touch(t, filepath.Join(dir, LegacyDatabaseName))
	path, err = ResolvePath(dir)
	if err != nil || path != filepath.Join(dir, LegacyDatabaseName) {
		t.Fatalf("legacy install: %q %v", path, err)
	}

	// Both files exist, so opening either one could silently pick wrong data.
	touch(t, filepath.Join(dir, DatabaseName))
	if _, err = ResolvePath(dir); err == nil {
		t.Fatal("ambiguous database files accepted")
	}
}

func TestResolvePathIgnoresDirectoryNamedLikeLegacyDatabase(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, LegacyDatabaseName), 0700); err != nil {
		t.Fatal(err)
	}
	path, err := ResolvePath(dir)
	if err != nil || path != filepath.Join(dir, DatabaseName) {
		t.Fatalf("directory treated as a database: %q %v", path, err)
	}
}

func TestV1UpgradePreservesDataAndBacksUp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, DatabaseName)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(schema); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`PRAGMA user_version=1;INSERT INTO user(id,username)VALUES(1,'old');INSERT INTO wrong_question(id,user_id,subject,question_core,semantic_summary,mastery_status)VALUES(1,1,'math','cos x','','mastered');`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	db, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var text, status string
	var due int64
	err = db.QueryRow(`SELECT q.search_text,q.mastery_status,p.due_at FROM wrong_question q JOIN review_plan p ON p.question_id=q.id WHERE q.id=1`).Scan(&text, &status, &due)
	if err != nil || text != "cosx" || status != "mastered" || due < time.Now().Add(6*24*time.Hour).Unix() {
		t.Fatal(text, status, due, err)
	}
	backups, _ := filepath.Glob(filepath.Join(dir, "before-upgrade-*.db"))
	if len(backups) != 1 {
		t.Fatal("backup missing")
	}
	if err = migrate(db, dir); err != nil {
		t.Fatal("migration not idempotent", err)
	}
}

func TestV2ClassificationMigrationPreservesLegacyAndQuarantinesVectors(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, DatabaseName))
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{schema, migrationV2, `PRAGMA user_version=2;INSERT INTO user(id,username)VALUES(1,'legacy');INSERT INTO wrong_question(id,user_id,subject,chapter,question_core,semantic_summary,mistake_summary)VALUES(1,1,'math','原章节','物理题','旧数学摘要','旧错因');INSERT INTO local_vector(question_id,vector_type,model,dimension,content_hash,vector)VALUES(1,'semantic','old',1,'hash',x'00000000');`} {
		if _, err = db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()
	db, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var subject, chapter, core, summary, mistake, id, status string
	var stale bool
	err = db.QueryRow(`SELECT subject,chapter,question_core,semantic_summary,mistake_summary,subject_id,classification_status,analysis_stale FROM wrong_question WHERE id=1`).Scan(&subject, &chapter, &core, &summary, &mistake, &id, &status, &stale)
	if err != nil || subject != "math" || chapter != "原章节" || core != "物理题" || summary != "旧数学摘要" || mistake != "旧错因" || id != "" || status != "legacy_pending" || !stale {
		t.Fatal("legacy content or classification changed", err, subject, chapter, status)
	}
	var n int
	db.QueryRow("SELECT count(*) FROM local_vector").Scan(&n)
	if n != 0 {
		t.Fatal("legacy vectors remain")
	}
	if err = migrate(db, dir); err != nil {
		t.Fatal(err)
	}
	backups, _ := filepath.Glob(filepath.Join(dir, "before-upgrade-*.db"))
	if len(backups) != 1 {
		t.Fatal("backup missing or repeated", backups)
	}
}

func TestV3UpgradeDefaultsStageWithoutReclassifying(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, DatabaseName))
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{schema, migrationV2, migrationV3, `PRAGMA user_version=3;INSERT INTO user(id,username) VALUES(1,'existing');INSERT INTO wrong_question(id,user_id,subject,subject_id,classification_status,question_core,semantic_summary) VALUES(1,1,'408','cs408','confirmed','原题','摘要');INSERT INTO custom_subject(id,user_id,name,normalized_name) VALUES('custom_old',1,'材料力学','材料力学');`} {
		if _, err = db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()
	db, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var stage, subject, status, customStage string
	if err = db.QueryRow("SELECT u.education_stage,q.subject_id,q.classification_status FROM user u JOIN wrong_question q ON q.user_id=u.id WHERE u.id=1").Scan(&stage, &subject, &status); err != nil {
		t.Fatal(err)
	}
	if stage != "university" || subject != "cs408" || status != "confirmed" {
		t.Fatal(stage, subject, status)
	}
	if err = db.QueryRow("SELECT education_stage FROM custom_subject WHERE id='custom_old'").Scan(&customStage); err != nil || customStage != "university" {
		t.Fatal(customStage, err)
	}
	if _, err = db.Exec("UPDATE user SET education_stage='highschool' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	if err = migrate(db, dir); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow("SELECT education_stage FROM user WHERE id=1").Scan(&stage); err != nil || stage != "highschool" {
		t.Fatal("stage reset on restart", stage, err)
	}
	backups, _ := filepath.Glob(filepath.Join(dir, "before-upgrade-*.db"))
	if len(backups) != 1 {
		t.Fatal(backups)
	}
}
