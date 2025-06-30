package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

       _ "modernc.org/sqlite"

	"github.com/lenhattri/kaeshi-migrate/pkg/validate"
)

// Dialect implements validate.Dialect for SQLite.
type Dialect struct{}

func (Dialect) DriverName() string { return "sqlite" }

func (Dialect) SplitStatements(input string) ([]string, error) { return validate.GenericSplit(input) }

func (Dialect) ParseBlocks(stmts []string) ([][]string, error) {
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
	ddl := map[string]bool{"CREATE": true, "ALTER": true, "DROP": true}
	switch {
	case dml[first]:
		return "DML"
	case ddl[first]:
		return "DDL"
	default:
		return "UNKNOWN"
	}
}

func (Dialect) IsCheckable(stmt string) bool { return true }

func (Dialect) IsSafeInTxn(stmt string) bool { return true }

func (Dialect) ValidateStmt(tx *sql.Tx, stmt string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, err := tx.ExecContext(ctx, stmt)
	return err
}
