package config

// Config represents application configuration loaded from file or environment.
type Config struct {
	Env      string `mapstructure:"env" yaml:"env"`
	User     string `mapstructure:"user" yaml:"user"`
	Database struct {
		Driver string `mapstructure:"driver" yaml:"driver"`
		Dsn    string `mapstructure:"dsn" yaml:"dsn"`
	} `mapstructure:"database" yaml:"database"`
	Logging struct {
		Level  string `mapstructure:"level" yaml:"level"`
		Driver string `mapstructure:"driver" yaml:"driver"`
		File   string `mapstructure:"file" yaml:"file"`
		Kafka  struct {
			Brokers []string `mapstructure:"brokers" yaml:"brokers"`
			Topic   string   `mapstructure:"topic" yaml:"topic"`
		} `mapstructure:"kafka" yaml:"kafka"`
		RabbitMQ struct {
			URL   string `mapstructure:"url" yaml:"url"`
			Queue string `mapstructure:"queue" yaml:"queue"`
		} `mapstructure:"rabbitmq" yaml:"rabbitmq"`
	} `mapstructure:"logging" yaml:"logging"`
}
