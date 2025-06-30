package manager

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/lenhattri/kaeshi-migrate/pkg/validate"
)

var (
	migrationDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "migration_duration_seconds",
		Help:    "Duration of migration operations",
		Buckets: prometheus.DefBuckets,
	})
	migrationsApplied = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "migrations_applied_total",
		Help: "Total number of migrations applied",
	})
	migrationsRollback = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "migrations_rollback_total",
		Help: "Total number of migrations rolled back",
	})
)

func init() {
	prometheus.MustRegister(migrationDuration, migrationsApplied, migrationsRollback)
}

// Manager wraps golang-migrate with retries, metrics, logging, and resource handling.
type Manager struct {
	m             *migrate.Migrate
	db            *sql.DB
	maxRetries    int
	migrationsDir string
	logger        *logrus.Entry
	actor         string // user performing the migration
	strictHash    bool
	dsn           string
	backend       DBBackend
	validateOpts  validate.ValidateOptions
}

// NewManager creates a Manager. It limits DB pool to 1 connection to ensure advisory locks
// (used internally by the Postgres driver) apply correctly.
func NewManager(backend DBBackend, dsn, migrationsDir string, retries int, logger *logrus.Entry, actor string, strict bool, confirmFn validate.ConfirmFunc) (*Manager, error) {
	db, err := sql.Open(backend.DriverName(), dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// Ensure only one open connection so Postgres advisory lock is effective.
	db.SetMaxOpenConns(2)
	db.SetConnMaxIdleTime(5 * time.Minute)

	driver, err := backend.NewDriver(db)
	if err != nil {
		return nil, fmt.Errorf("prepare migrate driver: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsDir,
		backend.DriverName(),
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("new migrate instance: %w", err)
	}

	return &Manager{
		m:             m,
		db:            db,
		maxRetries:    retries,
		migrationsDir: migrationsDir,
		logger:        logger,
		actor:         actor,
		strictHash:    strict,
		dsn:           dsn,
		backend:       backend,
		validateOpts: validate.ValidateOptions{
			SkipOnConfirmation: true,
			ConfirmFn:          confirmFn,
		},
	}, nil
}

// Close cleans up resources.
func (mgr *Manager) Close() error {
	_ = mgr.db.Close()
	err1, err2 := mgr.m.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

// CommitAll marks all rows in migrations_history as committed.
func (mgr *Manager) CommitAll() error {
	tx, err := mgr.db.Begin()
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE migrations_history SET committed = true WHERE committed = false`); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// versionCommitted reports whether the given version has been committed.
func (mgr *Manager) VersionCommitted(v uint) (bool, error) {
	var committed bool
	err := mgr.db.QueryRow(`SELECT committed FROM migrations_history WHERE version = $1 ORDER BY id DESC LIMIT 1`, fmt.Sprintf("%d", v)).Scan(&committed)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return committed, nil
}

// recordHistory inserts an entry into migrations_history for auditing.
func (mgr *Manager) recordHistory(action string, version uint) {
	actor := mgr.actor
	if actor == "" {
		actor = "unknown"
	}
	_, err := mgr.db.Exec(
		"INSERT INTO migrations_history(action, version, executed_by, committed) VALUES ($1,$2,$3,$4)",
		action, fmt.Sprintf("%d", version), actor, false,
	)
	if err != nil {
		mgr.logger.WithError(err).Warn("failed to record history")
	}
}

// withRetry retries the given migration operation up to maxRetries times.
func (mgr *Manager) withRetry(op func() error) error {
	var err error
	for attempt := 0; attempt <= mgr.maxRetries; attempt++ {
		if attempt > 0 {
			mgr.logger.WithField("attempt", attempt).
				Warn("retrying migration operation")
			time.Sleep(time.Second * time.Duration(attempt))
		}
		err = op()
		if err == nil || errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		mgr.logger.WithFields(logrus.Fields{
			"attempt": attempt,
			"error":   err,
		}).Error("migration operation failed")
	}
	mgr.logger.WithFields(logrus.Fields{
		"maxRetries": mgr.maxRetries,
		"error":      err,
	}).Error("all migration retries exhausted")
	return err
}

// pendingUpFiles returns all .up.sql files whose version is > current.
func (mgr *Manager) pendingUpFiles(cur uint) ([]string, error) {
	pattern := filepath.Join(mgr.migrationsDir, "*.up.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	var out []string
	for _, f := range files {
		parts := strings.SplitN(filepath.Base(f), "_", 2)
		if v, err := strconv.ParseUint(parts[0], 10, 64); err == nil && uint(v) > cur {
			out = append(out, f)
		}
	}
	return out, nil
}

// pendingDownFiles returns all .down.sql files for the given version, in reverse order.
func (mgr *Manager) pendingDownFiles(cur uint) ([]string, error) {
	pattern := filepath.Join(mgr.migrationsDir, fmt.Sprintf("%d_*.down.sql", cur))
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	return files, nil
}

func (mgr *Manager) Up() error {
	before, dirty, err := mgr.m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("read version before Up: %w", err)
	}
	if dirty {
		return fmt.Errorf("database dirty at version %d; manual intervention required", before)
	}

	// Lấy danh sách file up sẽ được apply (pending > before)
	upFiles, _ := mgr.pendingUpFiles(before)
	if len(upFiles) == 0 {
		mgr.logger.WithField("actor", mgr.actor).Info("no pending migrations to apply (Up)")
		return nil
	}

	// 1. Chặn file có version <= DB version
	for _, f := range upFiles {
		base := filepath.Base(f)
		parts := strings.SplitN(base, "_", 2)
		v, _ := strconv.ParseUint(parts[0], 10, 64)
		if uint(v) <= before {
			return fmt.Errorf(
				"migration version %d (file %s) is less than or equal to current DB version %d; refusing to apply, please rebase or resequence your migrations",
				v, base, before)
		}
		committed, err := mgr.VersionCommitted(uint(v))
		if err != nil {
			return err
		}
		if committed {
			return fmt.Errorf("migration version %d has been committed; cannot modify committed migrations", v)
		}
	}

	// 2. Check conflict hash cho các file version đã có trong history (phòng trường hợp rollback hoặc file copy lỗi)
	if mgr.strictHash {
		for _, f := range upFiles {
			base := filepath.Base(f)
			parts := strings.SplitN(base, "_", 2)
			v, _ := strconv.ParseUint(parts[0], 10, 64)
			hash, herr := fileHash(f)
			if herr != nil {
				return fmt.Errorf("cannot compute hash for %s: %v", f, herr)
			}
			//kiểm tra hash trong DB (nếu có)
			var dbHash string
			err := mgr.db.QueryRow(`SELECT sha256 FROM migrations_history WHERE action='up' AND version=$1 AND committed=true ORDER BY id DESC LIMIT 1`, fmt.Sprintf("%d", v)).Scan(&dbHash)
			if err == sql.ErrNoRows {
				continue
			}
			if err != nil {
				return fmt.Errorf("query hash: %w", err)
			}
			if dbHash != "" && dbHash != hash {
				return fmt.Errorf(
					"migration version %d (file %s) has been applied with a different hash; refusing to apply: current hash: %s, DB hash: %s; please fix the conflict",
					v, base, hash, dbHash)
			}
		}
	}

	// 3. Log filenames sắp apply
	for _, f := range upFiles {
		mgr.logger.WithField("actor", mgr.actor).Debugf("Applying migration file: %s", filepath.Base(f))

		data, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		content := string(data)
		fmt.Println(strings.TrimSpace(content))
		if ok, err := validate.ValidateSQL(content, map[string]string{"dsn": mgr.dsn}, mgr.validateOpts, mgr.backend.Validator()); !ok || err != nil {
			if err != nil {
				mgr.logger.WithError(err).Error("SQL validation failed")
			}
			return fmt.Errorf("invalid SQL in %s", filepath.Base(f))
		}
	}

	// 4. Thực thi migrate Up
	start := time.Now()
	err = mgr.withRetry(mgr.m.Up)
	migrationDuration.Observe(time.Since(start).Seconds())
	after, dirtyAfter, _ := mgr.m.Version()

	// 5. Ghi lại history với hash từng file vừa apply (từ before+1 đến after)
	if err == nil && after > before {
		for _, f := range upFiles {
			base := filepath.Base(f)
			parts := strings.SplitN(base, "_", 2)
			v, _ := strconv.ParseUint(parts[0], 10, 64)
			if uint(v) > before && uint(v) <= after {
				hash, herr := fileHash(f)
				if herr != nil {
					mgr.logger.WithError(herr).Warnf("cannot compute hash for %s", f)
				}
				actor := mgr.actor
				if actor == "" {
					actor = "unknown"
				}
				_, err := mgr.db.Exec(
					`INSERT INTO migrations_history(action, version, executed_by, sha256, committed) VALUES ($1,$2,$3,$4,$5)`,
					"up", fmt.Sprintf("%d", v), actor, hash, false)
				if err != nil {
					mgr.logger.WithError(err).Warnf("failed to record history with hash for version %d", v)
				} else {
					mgr.logger.WithFields(logrus.Fields{
						"version": v,
						"file":    base,
						"actor":   actor,
						"hash":    hash,
					}).Info("migration up applied and recorded")
				}
			}
		}
	}

	switch {
	case err != nil:
		mgr.logger.WithError(err).
			WithFields(logrus.Fields{"from": before, "to": after, "actor": mgr.actor}).
			Error("Up migration failed")
		return err
	case dirtyAfter:
		return fmt.Errorf("Up migration left database dirty at version %d", after)
	}
	return nil
}

// Down rolls back all applied migrations.
func (mgr *Manager) Down() error {
	before, dirty, err := mgr.m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("read version before Down: %w", err)
	}
	if dirty {
		return fmt.Errorf("database dirty at version %d; manual intervention required", before)
	}

	var exists bool
	if err := mgr.db.QueryRow(`SELECT true FROM migrations_history WHERE committed = true LIMIT 1`).Scan(&exists); err != nil && err != sql.ErrNoRows {
		return err
	}
	if exists {
		return fmt.Errorf("migration version %d has been committed; cannot modify committed migrations", before)
	}

	// Log filenames in reverse order
	if files, _ := mgr.pendingDownFiles(before); len(files) > 0 {
		for _, f := range files {
			mgr.logger.Debugf("Rolling back migration file: %s", filepath.Base(f))
		}
	}

	start := time.Now()
	err = mgr.withRetry(mgr.m.Down)
	migrationDuration.Observe(time.Since(start).Seconds())

	after, dirtyAfter, _ := mgr.m.Version()
	switch {
	case err != nil:
		mgr.logger.WithError(err).
			WithField("actor", mgr.actor).
			Error("Down migration failed")
		return err
	case dirtyAfter:
		return fmt.Errorf("Down migration left database dirty at version %d", after)
	case before > after:
		mgr.logger.WithFields(logrus.Fields{
			"from":  before,
			"to":    after,
			"actor": mgr.actor,
		}).Info("migrations rolled back (Down)")
		migrationsRollback.Add(float64(before - after))
		mgr.recordHistory("down", after)
	default:
		mgr.logger.WithField("actor", mgr.actor).Info("no migrations to roll back (Down)")
	}
	return nil
}

// Steps migrates exactly n steps (negative to rollback).
func (mgr *Manager) Steps(n int) error {
	before, dirty, err := mgr.m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("read version before Steps: %w", err)
	}
	if dirty {
		return fmt.Errorf("database dirty at version %d; manual intervention required", before)
	}

	if n < 0 {
		committed, err := mgr.VersionCommitted(before)
		if err != nil {
			return err
		}
		if committed {
			return fmt.Errorf("migration version %d has been committed; cannot modify committed migrations", before)
		}
	}

	if n < 0 {
		files, _ := mgr.pendingDownFiles(before)
		if len(files) > 0 {
			f := files[0]
			data, err := os.ReadFile(f)
			if err != nil {
				return fmt.Errorf("read %s: %w", f, err)
			}
			content := string(data)
			fmt.Println(strings.TrimSpace(content))
			if ok, err := validate.ValidateSQL(content, map[string]string{"dsn": mgr.dsn}, mgr.validateOpts, mgr.backend.Validator()); !ok || err != nil {
				if err != nil {
					mgr.logger.WithError(err).Error("SQL validation failed")
				}
				return fmt.Errorf("invalid SQL in %s", filepath.Base(f))
			}
		}
	}

	start := time.Now()
	err = mgr.withRetry(func() error { return mgr.m.Steps(n) })
	migrationDuration.Observe(time.Since(start).Seconds())

	after, dirtyAfter, _ := mgr.m.Version()
	switch {
	case err != nil:
		return err
	case dirtyAfter:
		return fmt.Errorf("Steps(%d) left database dirty at version %d", n, after)
	case after > before:
		mgr.logger.WithFields(logrus.Fields{
			"from":  before,
			"to":    after,
			"actor": mgr.actor,
		}).Infof("migrations applied %d steps", n)
		migrationsApplied.Add(float64(after - before))
		mgr.recordHistory("up", after)
	case before > after:
		mgr.logger.WithFields(logrus.Fields{
			"from":  before,
			"to":    after,
			"actor": mgr.actor,
		}).Infof("migrations rolled back %d steps", -n)
		migrationsRollback.Add(float64(before - after))
		mgr.recordHistory("rollback", after)
	default:
		mgr.logger.WithField("actor", mgr.actor).Info("no effect from Steps migration")
	}
	return nil
}

// Force sets the DB to a specific version and clears the dirty flag.
func (mgr *Manager) Force(version int) error {
	if err := mgr.m.Force(version); err != nil {
		return fmt.Errorf("force to version %d failed: %w", version, err)
	}
	mgr.logger.WithFields(logrus.Fields{
		"version": version,
		"actor":   mgr.actor,
	}).Warn("forced database version; dirty flag cleared")
	return nil
}

// Status returns (currentVersion, pendingCount, error).
func (mgr *Manager) Status() (uint, int, error) {
	ver, dirty, err := mgr.m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return 0, 0, err
	}
	files, _ := filepath.Glob(filepath.Join(mgr.migrationsDir, "*.up.sql"))
	pending := 0
	for _, f := range files {
		parts := strings.SplitN(filepath.Base(f), "_", 2)
		if v, e := strconv.Atoi(parts[0]); e == nil && uint(v) > ver {
			pending++
		}
	}
	if dirty {
		mgr.logger.WithFields(logrus.Fields{
			"version": ver,
			"actor":   mgr.actor,
		}).Warn("database is in dirty state")
	}
	return ver, pending, nil
}

// Version returns (currentVersion, dirtyFlag, error).
func (mgr *Manager) Version() (uint, bool, error) {
	return mgr.m.Version()
}

// SafeForce only allows forcing down by one if dirty, and never up beyond last file.
func (mgr *Manager) SafeForce(target int) error {
	cur, dirty, err := mgr.m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("cannot read version: %w", err)
	}
	committed, err := mgr.VersionCommitted(uint(target))
	if err != nil {
		return err
	}
	if committed {
		return fmt.Errorf("migration version %d has been committed; cannot modify committed migrations", target)
	}
	last, err := mgr.lastFileVersion()
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	if uint(target) > last {
		return fmt.Errorf("target version %d exceeds the last migration file %d", target, last)
	}
	if !dirty {
		return fmt.Errorf("database is NOT dirty (version %d); refusing to force", cur)
	}
	if uint(target) != cur-1 {
		return fmt.Errorf("dirty at %d; only allowed force to %d", cur, cur-1)
	}
	if err := mgr.m.Force(target); err != nil {
		return fmt.Errorf("force failed: %w", err)
	}
	mgr.logger.WithFields(logrus.Fields{
		"from":  cur,
		"to":    target,
		"actor": mgr.actor,
	}).Warn("SAFE-FORCE executed, dirty cleared")
	mgr.recordHistory("safe-force", uint(target))
	return nil
}

// lastFileVersion finds the highest version number among *.up.sql files.
func (mgr *Manager) lastFileVersion() (uint, error) {
	pattern := filepath.Join(mgr.migrationsDir, "*.up.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return 0, err
	}
	var max uint
	for _, f := range files {
		if v, e := strconv.ParseUint(strings.SplitN(filepath.Base(f), "_", 2)[0], 10, 64); e == nil {
			if uint(v) > max {
				max = uint(v)
			}
		}
	}
	return max, nil
}
