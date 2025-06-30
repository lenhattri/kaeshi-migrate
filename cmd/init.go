package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/lenhattri/kaeshi-migrate/internal/templates"
)

// NewInitCmd returns a command that creates config and migration templates.
func NewInitCmd() *cobra.Command {
	var cfgPath string
	var migrationsDir string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate config file and migrations directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfgPath == "" {
				cfgPath = "configs/config.yml"
			}
			if migrationsDir == "" {
				migrationsDir = "migrations"
			}
			if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
				return err
			}
			if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
				if err := os.WriteFile(cfgPath, []byte(templates.DefaultConfig), 0o644); err != nil {
					return err
				}
				cmd.Printf("created config at %s\n", cfgPath)
			} else if err == nil {
				cmd.Printf("config already exists at %s\n", cfgPath)
			} else {
				return err
			}

			if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
				return err
			}
			up := filepath.Join(migrationsDir, "000001_init.up.sql")
			down := filepath.Join(migrationsDir, "000001_init.down.sql")
			if _, err := os.Stat(up); os.IsNotExist(err) {
				if err := os.WriteFile(up, []byte(templates.InitUp), 0o644); err != nil {
					return err
				}
			}
			if _, err := os.Stat(down); os.IsNotExist(err) {
				if err := os.WriteFile(down, []byte(templates.InitDown), 0o644); err != nil {
					return err
				}
			}
			cmd.Printf("initialized migrations at %s\n", migrationsDir)
			return nil
		},
	}
	cmd.Flags().StringVar(&cfgPath, "config_path", "configs/config.yml", "path to config file")
	cmd.Flags().StringVar(&migrationsDir, "migrations", "migrations", "migrations directory")
	return cmd
}
