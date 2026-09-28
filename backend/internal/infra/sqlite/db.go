package sqlite

import (
	"database/sql"
	"database/sql/driver"
	_ "embed"
	"fmt"
	"github.com/kukusuyi/Questrace/backend/internal/pkg/searchtext"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	sqliteDriver "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

//go:embed migration_v2.sql
var migrationV2 string

//go:embed migration_v3.sql
var migrationV3 string

//go:embed migration_v4.sql
var migrationV4 string

const Version = 4

func init() {
	sqliteDriver.MustRegisterDeterministicScalarFunction("search_normalize", 1, func(_ *sqliteDriver.FunctionContext, args []driver.Value) (driver.Value, error) {
		if args[0] == nil {
			return "", nil
		}
		return searchtext.Normalize(fmt.Sprint(args[0])), nil
	})
}

const (
	// DatabaseName is the database file of a fresh installation.
	DatabaseName = "questrace.db"
	// LegacyDatabaseName is the pre-rename database file. An existing database
	// keeps being used in place so upgrades never copy or move user data.
	LegacyDatabaseName = "notebook.db"
)

// ResolvePath returns the database file a data directory uses. A directory that
// holds both the current and the legacy database is ambiguous, so it fails
// instead of silently opening the wrong data.
func ResolvePath(dir string) (string, error) {
	current := filepath.Join(dir, DatabaseName)
	legacy := filepath.Join(dir, LegacyDatabaseName)
	hasCurrent := fileExists(current)
	hasLegacy := fileExists(legacy)
	switch {
	case hasCurrent && hasLegacy:
		return "", fmt.Errorf("数据目录同时存在 %s 与 %s，请保留其中一个后重新启动", DatabaseName, LegacyDatabaseName)
	case hasLegacy:
		return legacy, nil
	default:
		return current, nil
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func Open(dir string) (*sql.DB, error) {
	path, err := ResolvePath(dir)
	if err != nil {
		return nil, err
	}
	u := databaseURL(path)
	db, err := sql.Open("sqlite", u.String()+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_time_format=sqlite")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err = migrate(db, dir); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
func migrate(db *sql.DB, dir string) error {
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version > Version {
		return fmt.Errorf("database version %d is newer than supported %d", version, Version)
	}
	if version == Version {
		return nil
	}
	if version > 0 {
		path := filepath.Join(dir, fmt.Sprintf("before-upgrade-%d.db", time.Now().UnixNano()))
		if _, err := db.Exec("VACUUM INTO '" + strings.ReplaceAll(path, "'", "''") + "'"); err != nil {
			return err
		}
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if version == 0 {
		if _, err = tx.Exec(schema); err != nil {
			return err
		}
		version = 1
	}
	if version < 2 {
		if _, err = tx.Exec(migrationV2); err != nil {
			return err
		}
	}
	if version < 3 {
		if _, err = tx.Exec(migrationV3); err != nil {
			return err
		}
	}
	if version < 4 {
		if _, err = tx.Exec(migrationV4); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(fmt.Sprintf("PRAGMA user_version=%d", Version)); err != nil {
		return err
	}
	return tx.Commit()
}

// A Windows drive must be in the URI path, never parsed as a hostname.
func databaseURL(path string) url.URL {
	path = filepath.ToSlash(path)
	if len(path) > 1 && path[1] == ':' {
		path = "/" + path
	}
	return url.URL{Scheme: "file", Path: path}
}
