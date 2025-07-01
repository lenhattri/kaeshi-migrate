package notifier

import "strings"

// Config defines notifier settings.
type Config struct {
	Enabled bool   `mapstructure:"enabled" yaml:"enabled"`
	Type    string `mapstructure:"type" yaml:"type"`
	Discord struct {
		WebhookURL string `mapstructure:"webhook_url" yaml:"webhook_url"`
	} `mapstructure:"discord" yaml:"discord"`
	Slack struct {
		WebhookURL string `mapstructure:"webhook_url" yaml:"webhook_url"`
	} `mapstructure:"slack" yaml:"slack"`
	Webhook struct {
		URL     string            `mapstructure:"url" yaml:"url"`
		Headers map[string]string `mapstructure:"headers" yaml:"headers"`
	} `mapstructure:"webhook" yaml:"webhook"`
}

// NewNotifier returns a Notifier implementation based on configuration.
func NewNotifier(cfg Config) Notifier {
	if !cfg.Enabled {
		return &NoopNotifier{}
	}
	switch strings.ToLower(cfg.Type) {
	case "discord":
		if cfg.Discord.WebhookURL != "" {
			return &DiscordNotifier{WebhookURL: cfg.Discord.WebhookURL}
		}
	case "slack":
		if cfg.Slack.WebhookURL != "" {
			return &SlackNotifier{WebhookURL: cfg.Slack.WebhookURL}
		}
	case "webhook":
		if cfg.Webhook.URL != "" {
			return &WebhookNotifier{URL: cfg.Webhook.URL, Headers: cfg.Webhook.Headers}
		}
	}
	return &NoopNotifier{}
}

func formatMessage(e MigrationEvent) string {
	msg := e.Status + " migration"
	if e.Version != "" {
		msg += " version " + e.Version
	}
	if e.DB != "" {
		msg += " on " + e.DB
	}
	if e.User != "" {
		msg += " by " + e.User
	}
	if e.Error != nil {
		msg += ": " + e.Error.Error()
	}
	return msg
}
