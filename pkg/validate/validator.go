package validate

import (
	"fmt"
	"strings"
	"time"
)

// ValidateSQL checks SQL syntax or safely executes it in a transaction without
// side-effects using the provided dialect.
func ValidateSQL(sqlText string, dbConfig map[string]string, opts ValidateOptions, d Dialect) (bool, error) {
	dsn, ok := dbConfig["dsn"]
	if !ok || strings.TrimSpace(dsn) == "" {
		return false, fmt.Errorf("dbConfig missing dsn")
	}

	if opts.Timeout == 0 {
		opts.Timeout = 4 * time.Second
	}

	trimmed := strings.TrimSpace(sqlText)
	if trimmed == "" {
		return false, fmt.Errorf("empty SQL statement")
	}
	if len(trimmed) > 100*1024 {
		return false, fmt.Errorf("SQL input too large")
	}

	stmts, err := d.SplitStatements(trimmed)
	if err != nil {
		return false, err
	}
	if len(stmts) == 0 {
		return false, fmt.Errorf("no statements found")
	}
	if len(stmts) > 100 {
		return false, fmt.Errorf("too many statements: %d", len(stmts))
	}

	blocks, err := d.ParseBlocks(stmts)
	if err != nil {
		return false, err
	}

	db, err := OpenDB(d.DriverName(), dsn)
	if err != nil {
		return false, err
	}
	defer db.Close()

	for _, b := range blocks {
		if err := validateBlock(db, b, opts, d); err != nil {
			return false, err
		}
	}
	return true, nil
}
