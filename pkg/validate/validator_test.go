package validate_test

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lenhattri/kaeshi-migrate/pkg/validate"
	"github.com/lenhattri/kaeshi-migrate/pkg/validate/postgres"
)

func withMockDB(t *testing.T, fn func(sqlmock.Sqlmock)) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("mock db: %v", err)
	}
	old := validate.OpenDB
	validate.OpenDB = func(driver, dsn string) (*sql.DB, error) { return db, nil }
	t.Cleanup(func() {
		validate.OpenDB = old
		db.Close()
	})
	fn(mock)
}

func TestDialectCheckable(t *testing.T) {
	d := postgres.Dialect{}
	if d.IsCheckable("DO $$BEGIN END$$;") {
		t.Fatalf("DO should not be checkable")
	}
	if !d.IsCheckable("SELECT 1") {
		t.Fatalf("SELECT should be checkable")
	}
}

func TestDialectSafeInTxn(t *testing.T) {
	d := postgres.Dialect{}
	if d.IsSafeInTxn("CREATE INDEX CONCURRENTLY idx ON t(id)") {
		t.Fatalf("concurrent index is unsafe")
	}
	if !d.IsSafeInTxn("CREATE TABLE a(id int)") {
		t.Fatalf("CREATE TABLE should be safe")
	}
}

func TestParseBlocksManual(t *testing.T) {
	d := postgres.Dialect{}
	stmts := []string{"BEGIN", "CREATE TABLE a(id int);", "INSERT INTO a VALUES(1);", "COMMIT", "SELECT 1;"}
	blocks, err := d.ParseBlocks(stmts)
	if err != nil {
		t.Fatalf("parseBlocks error: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if len(blocks[0]) != 2 || len(blocks[1]) != 1 {
		t.Fatalf("unexpected block lengths: %+v", blocks)
	}
}

func TestValidateSQLBlock(t *testing.T) {
	d := postgres.Dialect{}
	withMockDB(t, func(mock sqlmock.Sqlmock) {
		mock.ExpectBegin()
		mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("INSERT INTO").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()

		sqlText := "CREATE TABLE foo(id int); INSERT INTO foo VALUES(1);"
		ok, err := validate.ValidateSQL(sqlText, map[string]string{"dsn": "mock"}, validate.ValidateOptions{}, d)
		if err != nil || !ok {
			t.Fatalf("expected success, got ok=%v err=%v", ok, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("expectations: %v", err)
		}
	})
}

func TestValidateSQLConfirm(t *testing.T) {
	d := postgres.Dialect{}
	withMockDB(t, func(mock sqlmock.Sqlmock) {
		mock.ExpectBegin()
		mock.ExpectRollback()
		called := false
		ok, err := validate.ValidateSQL("VACUUM", map[string]string{"dsn": "mock"}, validate.ValidateOptions{
			SkipOnConfirmation: true,
			ConfirmFn: func(msg string) (bool, error) {
				t.Logf("confirm: %s", msg)
				called = true
				return true, nil
			},
		}, d)
		if !called {
			t.Fatal("confirm not called")
		}
		if err != nil || !ok {
			t.Fatalf("expected success, got ok=%v err=%v", ok, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("expectations: %v", err)
		}
	})
}
