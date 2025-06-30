package manager

import (
	"database/sql"

	"github.com/lenhattri/kaeshi-migrate/pkg/validate"
	"github.com/golang-migrate/migrate/v4/database"
)

// DBBackend abstracts database specific logic for migrations.
type DBBackend interface {
	DriverName() string
	NewDriver(db *sql.DB) (database.Driver, error)
	Validator() validate.Dialect
}

var backends = map[string]DBBackend{}

// RegisterBackend registers a backend implementation by name.
func RegisterBackend(name string, b DBBackend) {
	backends[name] = b
}

// GetBackend returns the backend registered for driver name.
func GetBackend(name string) (DBBackend, bool) {
	b, ok := backends[name]
	return b, ok
}
