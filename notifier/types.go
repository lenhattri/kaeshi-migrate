package notifier

import "time"

// Notifier interface for sending migration events.
type Notifier interface {
	Notify(event MigrationEvent) error
}

// MigrationEvent holds contextual data about a migration action.
type MigrationEvent struct {
	Status   string // success, fail, rollback, etc.
	User     string
	Version  string
	DB       string
	Duration time.Duration
	Error    error
	Time     time.Time
}
