# Kaeshi Migrate – Reliable and Auditable Database Migrations for Go

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Kaeshi** is a safe and auditable database migration tool for Go projects.  
It builds upon [golang-migrate](https://github.com/golang-migrate/migrate), adding features designed for operational safety, traceability, and team accountability.

---

## Dependencies

- Go 1.24.3 or newer
- PostgreSQL, MySQL, or SQLite (as your database)
- Kafka or RabbitMQ (optional, for advanced logging)
- Optional webhook notifications (Discord, Slack, generic)

---

## ✨ Key Features

- Generate migration files with automatic versioning and author tagging.
- Validate SQL syntax for PostgreSQL, MySQL, and SQLite before execution.
- Record all migration actions into `migrations_history`, including version, time, user, and hash.
- Detect and prevent unauthorized edits to committed migrations.
- Structured logging to file, stdout, Kafka, or RabbitMQ.
- Webhook notifications for migration events (Discord, Slack, generic).
- Built-in safeguards for dirty database states (`safe-force` only steps back).
- Integrated Makefile targets for smooth developer workflows.

---

## 📦 Getting Started
### 0. Install Go on Linux/macOS (optional)

```bash
curl -LO https://go.dev/dl/go1.24.3.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
````
### 1. Install

Clone and build:

```bash
git clone https://github.com/lenhattri/kaeshi-migrate.git
cd kaeshi-migrate
go mod tidy
go build -o kaeshi ./cmd/migrate
````

### 2. Configure

First generate a config file and migrations folder:

```bash
./kaeshi init --config_path ./configs/config.yml --migrations ./migrations
```

Edit the generated `configs/config.yml`:

```yaml
env: development
user: yourname

database:
  driver: postgres
  dsn: "postgres://user:pass@localhost:5432/db?sslmode=disable"

logging:
  level: info
  driver: kafka
  file: ""  # file logging in non-production

  kafka:
    brokers: ["localhost:9092"]
    topic: logging

  rabbitmq:
    url: "amqp://guest:guest@localhost:5672/"
    queue: logging

notifier:
  enabled: false
  type: webhook
  discord:
    webhook_url: ""
  slack:
    webhook_url: ""
  webhook:
    url: ""
    headers: {}
```

You can override any config with environment variables prefixed by `KAESHI_`.
For example: `KAESHI_DATABASE_DSN=...`

---

## 🚀 Usage

Build and explore the CLI:

```bash
./kaeshi help
```

### Core commands

| Command                | Description                                   |
| ---------------------- | --------------------------------------------- |
| `create [name]`        | Generate `.up.sql` and `.down.sql` files      |
| `up`                   | Apply all pending migrations                  |
| `down`                 | Roll back all migrations                      |
| `rollback`             | Roll back the most recent migration           |
| `status`               | View current version and pending migrations   |
| `version`              | Print current migration version               |
| `safe-force [version]` | Force rollback 1 step if DB is dirty          |
| `init`                 | Generate config file and migrations directory |
| `commit`               | Mark migrations as finalized and immutable    |

Flags:

* `--user yourname` to record the user who ran the command.
* `-y` / `--yes` to auto-confirm prompts.

---

## 🔧 Makefile Targets

Predefined targets are available for local development:

```bash
make migrate-up
make migrate-down
make migrate-status
make migrate-rollback
make migrate-create name=create_users user=yourname
make migrate-force version=20240624 user=yourname
```

---

## 📊 Observability & Logging

* **Notification Options**:
  * Discord webhook
  * Slack webhook
  * Generic webhook URL

* **Logging Options**:

  * Local file (for non-production)
  * Structured stdout
  * Kafka or RabbitMQ integration for centralized observability

---

## 🔒 Migration Commit Lock

Kaeshi supports a "commit" step to lock migration history and ensure consistency across teams:

```bash
kaeshi commit
```

Once committed, future `up` attempts will respect the locked state and prevent unauthorized reapplication or edits. This enforces safe, immutable infrastructure changes across environments.

---

## 👥 Contributing

Kaeshi welcomes contributions from developers who value safety and clarity in database operations.

* Please submit clean pull requests with accompanying tests.
* Run `go test ./...` before submitting.
* For questions, feedback, or collaboration, feel free to open an issue or start a discussion.

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).
You are free to use, modify, and distribute it with attribution.

---

## 💡 Project Vision

Database migrations are critical infrastructure changes. They deserve clear history, validation, and observability — not guesswork or manual patching.

Kaeshi aims to bring **discipline and clarity** to schema changes in Go projects, enabling teams to move fast **without breaking production**.

If you share that mindset, we’d love your feedback and contributions.


