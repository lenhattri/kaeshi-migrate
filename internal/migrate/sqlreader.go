package migration

import (
	"os"
	"strings"
)

// readStatements returns trimmed SQL statements separated by semicolons.
func readStatements(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(data), ";")
	var stmts []string
	for _, p := range parts {
		stmt := strings.TrimSpace(p)
		if stmt != "" {
			stmts = append(stmts, stmt+";")
		}
	}
	return stmts, nil
}
