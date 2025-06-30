package mysql

import (
	"context"
	"database/sql"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/lenhattri/kaeshi-migrate/pkg/validate"
)

// Dialect implements validate.Dialect for MySQL.
type Dialect struct{}

func (Dialect) DriverName() string { return "mysql" }

func (Dialect) SplitStatements(input string) ([]string, error) { return validate.GenericSplit(input) }

func (Dialect) ParseBlocks(stmts []string) ([][]string, error) {
	// MySQL does not support transactional DDL in the same way. Treat each statement as its own block.
	blocks := make([][]string, len(stmts))
	for i, s := range stmts {
		blocks[i] = []string{s}
	}
	if len(blocks) == 0 {
		blocks = append(blocks, []string{})
	}
	return blocks, nil
}

func (Dialect) StatementType(stmt string) string {
	if stmt == "" {
		return "UNKNOWN"
	}
	first := strings.ToUpper(strings.Fields(stmt)[0])
	dml := map[string]bool{"SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true}
	ddl := map[string]bool{"CREATE": true, "ALTER": true, "DROP": true, "TRUNCATE": true, "RENAME": true}
	switch {
	case dml[first]:
		return "DML"
	case ddl[first]:
		return "DDL"
	default:
		return "UNKNOWN"
	}
}

func (Dialect) IsCheckable(stmt string) bool {
	up := strings.ToUpper(strings.TrimSpace(stmt))
	if strings.HasPrefix(up, "DELIMITER") {
		return false
	}
	return true
}

func (Dialect) IsSafeInTxn(stmt string) bool {
	// Assume most statements are safe except explicit operations known to be unsafe.
	up := strings.ToUpper(strings.TrimSpace(stmt))
	if strings.HasPrefix(up, "CREATE DATABASE") || strings.HasPrefix(up, "DROP DATABASE") {
		return false
	}
	return true
}

func (Dialect) ValidateStmt(tx *sql.Tx, stmt string, timeout time.Duration) error {
	typ := Dialect{}.StatementType(stmt)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if typ == "DML" {
		_, err := tx.ExecContext(ctx, "EXPLAIN "+stmt)
		return err
	}
	_, err := tx.ExecContext(ctx, stmt)
	return err
}
