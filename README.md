# Kaeshi

`kaeshi-migration` is a command line tool for managing database schema changes. It wraps [golang-migrate](https://github.com/golang-migrate/migrate) with extra safety features such as SQL validation, migration history and structured logging.

## Features

- Generate new migration files with automatic version numbers and author information.
- Validate SQL statements for PostgreSQL, MySQL and SQLite before executing.
- Record every migration action in `migrations_history` including version, time, user and optional SHA256 hash.
- Detect conflicting hashes when re-applying an existing version.
- Structured logging to stdout, file and optionally Kafka or RabbitMQ.
- Prometheus counters and histograms for applied and rolled back migrations.
- Safe "force" mode that only allows forcing the database one version backwards when dirty.
- Convenient Makefile targets for common tasks.

## Installation

```bash
go mod tidy
```

Edit `configs/config.yml` for your environment:

```yaml
env: development
user: yourname

database:
  driver: postgres
  dsn: "postgres://user:pass@host:5432/db?sslmode=disable"

logging:
  level: info
  driver: kafka           # kafka | rabbitmq
  file: ""                # log file when env != production
  kafka:
    brokers: ["localhost:9092"]
    topic: logging
  rabbitmq:
    url: "amqp://guest:guest@localhost:5672/"
    queue: logging
```

Environment variables prefixed with `KAESHI_` override the file values. For example `KAESHI_DATABASE_DSN`.

## CLI Usage

Build and run using Go:

```bash
go build -o kaeshi-migrate ./cmd/migrate
./kaeshi help
```

Main commands:

- `create [name]` – generate new `.up.sql` and `.down.sql` files.
- `up` – apply all pending migrations.
- `down` – roll back all migrations.
- `rollback` – roll back the last migration only.
- `status` – show current version and pending migrations.
- `version` – print the current migration version.
- `safe-force [version]` – when the database is dirty, force it back one version.
- `commit` – mark all applied migrations as committed

Use `--user yourname` to record who executed the command. The flag `-y`/`--yes` automatically answers "yes" to confirmation prompts.

## Makefile shortcuts

The repository provides several make targets for convenience:

```bash
make migrate-up            # apply migrations
make migrate-down          # rollback all
make migrate-status        # show status
make migrate-rollback      # rollback one step
make migrate-create name=add_table user=you   # generate files
make migrate-force version=N user=you         # safe-force when dirty
```

## Typical workflow

1. Pull the latest code and run `make migrate-up` to ensure your database is up to date.
2. Create a new migration: `make migrate-create name=description user=you`.
3. Edit the generated SQL files.
4. Test the migration locally using `make migrate-up` and `make migrate-rollback`.
5. Commit the migration files and push to the repository.
6. Lock applied migrations:
   ```bash
   kaeshi commit
   ```
   Running `kaeshi up` on a committed version will return:
   `migration version X has been committed; cannot modify committed migrations`.

## Contributing

Issues and pull requests are welcome. Please run `go test ./...` before submitting changes.

