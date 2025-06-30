package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"github.com/lenhattri/kaeshi-migrate/pkg/validate"
)

// Dialect implements validate.Dialect for PostgreSQL.
type Dialect struct{}

func (Dialect) DriverName() string { return "postgres" }

func (Dialect) SplitStatements(input string) ([]string, error) { return validate.GenericSplit(input) }

func (Dialect) ParseBlocks(stmts []string) ([][]string, error) {
	var blocks [][]string
	var cur []string
	inBlock := false

	for _, s := range stmts {
		up := strings.ToUpper(strings.TrimSpace(strings.TrimSuffix(s, ";")))
		switch up {
		case "BEGIN", "BEGIN TRANSACTION", "START TRANSACTION":
			if inBlock {
				return nil, fmt.Errorf("nested BEGIN not allowed")
			}
			if len(cur) > 0 {
				blocks = append(blocks, cur)
				cur = nil
			}
			inBlock = true
			continue
		case "COMMIT", "END", "ROLLBACK":
			if !inBlock {
				return nil, fmt.Errorf("COMMIT without BEGIN")
			}
			blocks = append(blocks, cur)
			cur = nil
			inBlock = false
			continue
		}
		cur = append(cur, s)
	}
	if inBlock {
		return nil, fmt.Errorf("unterminated BEGIN block")
	}
	if len(cur) > 0 {
		blocks = append(blocks, cur)
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
	dml := map[string]bool{"SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true, "WITH": true}
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
	uncheck := []string{"DO", "COPY", "SET", "GRANT", "REVOKE"}
	for _, u := range uncheck {
		if strings.HasPrefix(up, u) {
			return false
		}
	}
	return true
}

func (Dialect) IsSafeInTxn(stmt string) bool {
	up := strings.ToUpper(strings.TrimSpace(stmt))
	nonTx := []string{
		"VACUUM",
		"CREATE DATABASE",
		"DROP DATABASE",
		"CREATE TABLESPACE",
		"DROP TABLESPACE",
		"CREATE INDEX CONCURRENTLY",
		"DROP INDEX CONCURRENTLY",
		"REINDEX",
		"CLUSTER",
		"ALTER SYSTEM",
		"REFRESH MATERIALIZED VIEW CONCURRENTLY",
	}
	for _, n := range nonTx {
		if strings.HasPrefix(up, n) {
			return false
		}
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
