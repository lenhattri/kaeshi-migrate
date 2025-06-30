package main

import (
	"database/sql"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	appcmd "github.com/lenhattri/kaeshi-migrate/cmd"
	"github.com/spf13/cobra"

	"github.com/lenhattri/kaeshi-migrate/internal/config"
	migration "github.com/lenhattri/kaeshi-migrate/internal/migrate"
	mgmt "github.com/lenhattri/kaeshi-migrate/internal/migrate/manager"
	"github.com/lenhattri/kaeshi-migrate/pkg/logger"
	"github.com/sirupsen/logrus"
)

func main() {
	// panic handler: luôn ghi log hoặc stdout cho stacktrace
	var log *logrus.Logger
	defer func() {
		if r := recover(); r != nil {
			if log != nil {
				log.WithFields(logrus.Fields{
					"component":   "panic",
					"error.stack": string(debug.Stack()),
				}).Errorf("panic: %v", r)
			} else {
				fmt.Fprintf(os.Stderr, "panic: %v\n%s", r, debug.Stack())
			}
			os.Exit(101)
		}
	}()

	rootCmd := appcmd.NewRootCmd()

	var (
		userFlag string
		cfg      *config.Config
		mgr      *mgmt.Manager
		backend  mgmt.DBBackend
	)

	rootCmd.PersistentFlags().StringVar(&userFlag, "user", "", "name executing the command")

	// initApp lazily loads configuration and initializes the manager
	initApp := func() error {
		if mgr != nil {
			return nil
		}
		var err error
		cfg, err = config.Load()
		if err != nil {
			return err
		}
		if userFlag == "" {
			userFlag = cfg.User
		}
		log = logger.New(
			cfg.Logging.Level,
			cfg.Env,
			cfg.Logging.Driver,
			cfg.Logging.Kafka.Brokers,
			cfg.Logging.Kafka.Topic,
			cfg.Logging.RabbitMQ.URL,
			cfg.Logging.RabbitMQ.Queue,
			cfg.Logging.File,
		)
		var ok bool
		backend, ok = mgmt.GetBackend(cfg.Database.Driver)
		if !ok {
			return fmt.Errorf("unknown database driver: %s", cfg.Database.Driver)
		}
		mgr, err = mgmt.NewManager(backend, cfg.Database.Dsn, "migrations", 3, log.WithField("component", "migrate"), userFlag, cfg.Env == "production", appcmd.AskConfirmation)
		if err != nil {
			return err
		}
		return nil
	}

	defer func() {
		if mgr != nil {
			_ = mgr.Close()
		}
	}()

	// ---- CREATE
	rootCmd.AddCommand(&cobra.Command{
		Use:   "create [name]",
		Short: "Generate new migration files",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initApp()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if userFlag == "" {
				return fmt.Errorf("--user or config.user is required")
			}
			db, err := sql.Open(backend.DriverName(), cfg.Database.Dsn)
			if err != nil {
				return err
			}
			defer db.Close()
			file, err := migration.Generate("migrations", args[0], userFlag, db)
			if err != nil {
				log.WithError(err).Error("generate migration file")
				return err
			}
			verStr := strings.SplitN(file, "_", 2)[0]
			ver, _ := strconv.ParseUint(verStr, 10, 64)
			committed, err := mgr.VersionCommitted(uint(ver))
			if err != nil {
				return err
			}
			if committed {
				return fmt.Errorf("migration version %d has been committed; cannot modify committed migrations", ver)
			}
			cmd.Println(file)
			return nil
		},
	})

	// ---- UP
	rootCmd.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "Apply all pending migrations",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initApp()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			err := mgr.Up()
			switch {
			case err == nil:
				cmd.Println("✅ Migrations applied successfully.")
				return nil
			case err == migrate.ErrNoChange:
				cmd.Println("✅ No new migrations to apply.")
				return nil
			default:
				log.WithError(err).Error("migration up failed")
				return err
			}
		},
	})

	// ---- DOWN
	rootCmd.AddCommand(&cobra.Command{
		Use:   "down",
		Short: "Rollback all migrations (danger: prod)",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initApp()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			err := mgr.Down()
			if err != nil {
				log.WithError(err).Error("migration down failed")
			}
			return err
		},
	})

	// ---- ROLLBACK
	rootCmd.AddCommand(&cobra.Command{
		Use:   "rollback",
		Short: "Rollback one migration step",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initApp()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			err := mgr.Steps(-1)
			if err != nil {
				log.WithError(err).Error("rollback step failed")
			}
			return err
		},
	})

	// ---- COMMIT
	rootCmd.AddCommand(&cobra.Command{
		Use:   "commit",
		Short: "Mark all applied migrations as committed",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initApp()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgr.CommitAll(); err != nil {
				log.WithError(err).Error("commit failed")
				return err
			}
			cmd.Println("✅ All applied migrations have been committed; strict hash checking is now enforced.")
			return nil
		},
	})

	// ---- STATUS
	rootCmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initApp()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v, pending, err := mgr.Status()
			if err != nil {
				log.WithError(err).Error("get status failed")
				return err
			}
			cmd.Printf("Current version: %d\nPending migrations: %d\n", v, pending)
			return nil
		},
	})

	// ---- VERSION
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print current migration version",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initApp()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v, dirty, err := mgr.Version()
			if err != nil {
				log.WithError(err).Error("get version failed")
				return err
			}
			cmd.Printf("Current version: %d", v)
			if dirty {
				cmd.Printf(" (DIRTY)")
			}
			cmd.Println()
			return nil
		},
	})

	// ---- SAFE-FORCE
	rootCmd.AddCommand(&cobra.Command{
		Use:   "safe-force [version]",
		Short: "Force to previous version only if dirty (Safe production use)",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initApp()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v, err := strconv.Atoi(args[0])
			if err != nil {
				log.WithError(err).Error("invalid version input")
				return fmt.Errorf("invalid version: %w", err)
			}
			err = mgr.SafeForce(v)
			if err != nil {
				log.WithError(err).Error("safe-force failed")
				return err
			}
			cmd.Printf("✅ Safe-forced database version to %d (dirty cleared)\n", v)
			return nil
		},
	})

	// ---- EXECUTE CLI
	if err := rootCmd.Execute(); err != nil {
		if strings.Contains(err.Error(), "unknown command") || strings.Contains(err.Error(), "unknown flag") {
			fmt.Fprintln(os.Stderr, "[CLI] "+err.Error())
			os.Exit(3)
		}
		fmt.Fprintln(os.Stderr, "[FATAL]", err.Error())
		os.Exit(2)
	}
}
