package confirm

import "fmt"

// ConfirmFunc is a user-provided callback for handling confirmations.
type ConfirmFunc func(prompt string) (bool, error)

// ErrConfirmRequired indicates manual confirmation is needed to proceed.
var ErrConfirmRequired = fmt.Errorf("confirmation required to skip automatic validation")

// FallbackConfirm prompts the user via provided ConfirmFunc when a statement
// cannot be automatically validated. It returns an error if confirmation fails.
func FallbackConfirm(fn ConfirmFunc, stmt, reason string) error {
	if fn == nil {
		return fmt.Errorf("%w: %s", ErrConfirmRequired, reason)
	}
	ok, err := fn(fmt.Sprintf("%s\n%s", reason, stmt))
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: %s", ErrConfirmRequired, reason)
	}
	return nil
}
