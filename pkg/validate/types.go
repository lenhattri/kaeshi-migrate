package validate

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lenhattri/kaeshi-migrate/pkg/validate/confirm"
)

// Dialect defines database dialect specific behaviors used during validation.
type Dialect interface {
	DriverName() string
	IsCheckable(stmt string) bool
	IsSafeInTxn(stmt string) bool
	SplitStatements(input string) ([]string, error)
	ParseBlocks(stmts []string) ([][]string, error)
	ValidateStmt(tx *sql.Tx, stmt string, timeout time.Duration) error
	StatementType(stmt string) string
}

// ErrConfirmRequired indicates manual confirmation is needed to proceed.
var ErrConfirmRequired = confirm.ErrConfirmRequired

// ConfirmFunc is a user-provided callback for handling confirmations.
type ConfirmFunc = confirm.ConfirmFunc

// LogLevel controls verbosity of validation logs.
type LogLevel int

const (
	LevelError LogLevel = iota
	LevelWarn
	LevelInfo
	LevelDebug
)

// ValidateOptions holds options controlling behavior of validation.
type ValidateOptions struct {
	SkipOnConfirmation bool
	ConfirmFn          ConfirmFunc
	Timeout            time.Duration
	LogLevel           LogLevel
}

// ValidationError provides details about a failed statement validation.
type ValidationError struct {
	Statement string
	Reason    string
	Err       error
	Type      string
}

func (e *ValidationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Type, e.Reason, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Reason)
}

// OpenDB abstracts sql.Open for testing.
var OpenDB = sql.Open
