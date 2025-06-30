package migration

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// nextVersion checks both the DB and filesystem to determine the next migration version number.
func nextVersion(db *sql.DB, dir string) (int, error) {
	maxDB := 0
	if db != nil {
		err := db.Ping()
		if err == nil {
			_ = db.QueryRow(`SELECT COALESCE(MAX(version::int),0) FROM schema_migrations`).Scan(&maxDB)
		}
	}

	maxFS := 0
	files, _ := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	for _, f := range files {
		base := filepath.Base(f)
		num := strings.SplitN(base, "_", 2)[0]
		v, _ := strconv.Atoi(num)
		if v > maxFS {
			maxFS = v
		}
	}

	if maxDB > maxFS {
		return maxDB + 1, nil
	}
	return maxFS + 1, nil
}

// Generate creates empty up and down SQL files with a unique next version number.
// The author will be recorded in the SQL comment header.
func Generate(path, name, author string, db *sql.DB) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	if author == "" {
		author = "unknown"
	}

	version, err := nextVersion(db, path)
	if err != nil {
		return "", err
	}

	baseName := fmt.Sprintf("%06d_%s", version, name)
	upFile := filepath.Join(path, baseName+".up.sql")
	downFile := filepath.Join(path, baseName+".down.sql")

	upContent := fmt.Sprintf("-- Author: %s\n-- Migration: %s\n-- Version: %06d\n\n-- Write your SQL here\n", author, name, version)
	downContent := fmt.Sprintf("-- Author: %s\n-- Migration: %s\n-- Version: %06d\n\n-- Write your SQL here\n", author, name, version)

	if err := os.WriteFile(upFile, []byte(upContent), 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(downFile, []byte(downContent), 0o644); err != nil {
		return "", err
	}
	return baseName, nil
}
