package manager

import (
	"database/sql"

	"github.com/golang-migrate/migrate/v4/database"
	mpostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/lib/pq"

	"github.com/lenhattri/kaeshi-migrate/pkg/validate"
	pgdialect "github.com/lenhattri/kaeshi-migrate/pkg/validate/postgres"
)

// PostgresBackend implements DBBackend for PostgreSQL databases.
type PostgresBackend struct{}

func (PostgresBackend) DriverName() string { return "postgres" }

func (PostgresBackend) NewDriver(db *sql.DB) (database.Driver, error) {
	return mpostgres.WithInstance(db, &mpostgres.Config{})
}

func (PostgresBackend) Validator() validate.Dialect { return pgdialect.Dialect{} }

func init() {
	RegisterBackend("postgres", PostgresBackend{})
}
