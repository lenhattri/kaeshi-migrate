# Kaeshi Migrate – Reliable and Auditable Database Migrations for Go

[![Go Report Card](https://goreportcard.com/badge/github.com/youruser/kaeshi)](https://goreportcard.com/report/github.com/youruser/kaeshi)
[![Build Status](https://github.com/youruser/kaeshi/actions/workflows/go.yml/badge.svg)](https://github.com/youruser/kaeshi/actions)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Kaeshi** is a safe and auditable database migration tool for Go projects.  
It builds upon [golang-migrate](https://github.com/golang-migrate/migrate), adding features designed for operational safety, traceability, and team accountability.

---

## ✨ Key Features

- Generate migration files with automatic versioning and author tagging.
- Validate SQL syntax for PostgreSQL, MySQL, and SQLite before execution.
- Record all migration actions into `migrations_history`, including version, time, user, and optional hash.
- Detect and prevent unauthorized edits to committed migrations.
- Structured logging to file, stdout, Kafka, or RabbitMQ.
- Prometheus metrics for observability in production environments.
- Built-in safeguards for dirty database states (`safe-force` only steps back).
- Integrated Makefile targets for smooth developer workflows.

---

## 📦 Getting Started

### 1. Install

Clone and build:

```bash
git clone https://github.com/youruser/kaeshi.git
cd kaeshi
go mod tidy
go build -o kaeshi-migrate ./cmd/migrate
````

### 2. Configure

Edit `configs/config.yml`:

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
```

You can override any config with environment variables prefixed by `KAESHI_`.
For example: `KAESHI_DATABASE_DSN=...`

---

## 🚀 Usage

Build and explore the CLI:

```bash
./kaeshi-migrate help
```

### Core commands

| Command                | Description                                 |
| ---------------------- | ------------------------------------------- |
| `create [name]`        | Generate `.up.sql` and `.down.sql` files    |
| `up`                   | Apply all pending migrations                |
| `down`                 | Roll back all migrations                    |
| `rollback`             | Roll back the most recent migration         |
| `status`               | View current version and pending migrations |
| `version`              | Print current migration version             |
| `safe-force [version]` | Force rollback 1 step if DB is dirty        |
| `commit`               | Mark migrations as finalized and immutable  |

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

* **Prometheus Metrics**:

  * `kaeshi_migrations_applied_total`
  * `kaeshi_migrations_rolledback_total`
  * `kaeshi_migration_duration_seconds`

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

