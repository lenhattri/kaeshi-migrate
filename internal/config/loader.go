package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Load reads configuration from the given file path and environment variables.
// If path is empty, it defaults to ./configs/config.yml.
// Environment variables prefixed with KAESHI_ take precedence.
func Load(path string) (*Config, error) {
	v := viper.New()
	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
	}
	v.AutomaticEnv()
	v.SetEnvPrefix("KAESHI")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Database.Dsn == "" {
		return nil, fmt.Errorf("database.dsn is required")
	}
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "postgres"
	}
	if cfg.Env == "" {
		cfg.Env = "development"
	}
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Driver == "" {
		cfg.Logging.Driver = "kafka"
	}
	if cfg.Logging.Kafka.Topic == "" {
		cfg.Logging.Kafka.Topic = "logging"
	}
	if cfg.Logging.File == "" && cfg.Env != "production" {
		cfg.Logging.File = "app.log"
	}

	return &cfg, nil
}
