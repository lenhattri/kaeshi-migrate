package validate

import (
	"database/sql"
	"strings"
	"time"

	"github.com/lenhattri/kaeshi-migrate/pkg/validate/confirm"
)

// validateBlock executes all statements in a block within a transaction and
// rolls back after validation.
func validateBlock(db *sql.DB, block []string, opts ValidateOptions, d Dialect) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, stmt := range block {
		trimmed := strings.TrimSpace(stmt)
		typ := d.StatementType(trimmed)

		if !d.IsCheckable(trimmed) {
			if opts.SkipOnConfirmation {
				if err := confirm.FallbackConfirm(opts.ConfirmFn, trimmed, "statement not automatically checkable"); err != nil {
					return &ValidationError{Statement: trimmed, Reason: "confirmation failed", Err: err, Type: typ}
				}
				continue
			}
			return &ValidationError{Statement: trimmed, Reason: "statement not automatically checkable", Err: ErrConfirmRequired, Type: typ}
		}

		if !d.IsSafeInTxn(trimmed) {
			if opts.SkipOnConfirmation {
				if err := confirm.FallbackConfirm(opts.ConfirmFn, trimmed, "cannot run in transaction"); err != nil {
					return &ValidationError{Statement: trimmed, Reason: "confirmation failed", Err: err, Type: typ}
				}
				continue
			}
			return &ValidationError{Statement: trimmed, Reason: "cannot run in transaction", Err: nil, Type: typ}
		}

		start := time.Now()
		if err := d.ValidateStmt(tx, trimmed, opts.Timeout); err != nil {
			return &ValidationError{Statement: trimmed, Reason: "execution failed", Err: err, Type: typ}
		}
		_ = start
	}
	return nil
}
