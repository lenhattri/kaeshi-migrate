package notifier

// NoopNotifier does nothing.
type NoopNotifier struct{}

func (n *NoopNotifier) Notify(event MigrationEvent) error { return nil }
