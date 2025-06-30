# Agent.md

## Project: kaeshi-migrate
**Go version**: 1.24.2  
**Module path**: `github.com/lenhattri/kaeshi-migrate`

### Overview  
This document defines the **Migration Agent** for building and managing schema migrations in a production-grade Go CLI named **kaeshi-migrate**. It covers project structure, configuration, commands, error handling, logging, CI/CD integration (GitLab CI), and best practices.

---

## 1. Project Layout  
```
kaeshi-migrate/
├── configs/
│   └── config.yaml            # DSN & environment settings
├── migrations/
│   ├── 000001_init.up.sql
│   ├── 000001_init.down.sql
│   └── …                      # additional versioned migrations
├── cmd/
│   └── migrate/
│       └── main.go            # CLI entrypoint
├── internal/
│   ├── config/
│   │   └── loader.go          # config loader
│   └── migration/
│       └── manager.go         # migration wrapper & utilities
├── pkg/
│   └── logger/
│       └── logger.go          # structured logging
├── go.mod                      # module definition
├── go.sum
├── .gitlab-ci.yml             # GitLab CI pipeline
└── README.md
```  

---

## 2. go.mod  
```go
module github.com/lenhattri/kaeshi-migrate

go 1.24.2

require (
    github.com/golang-migrate/migrate/v4 v4.15.4
    github.com/spf13/viper v1.15.0
    github.com/spf13/cobra v1.8.1
    github.com/sirupsen/logrus v1.9.0
    github.com/Shopify/sarama v1.36.0             // Kafka client
    github.com/prometheus/client_golang v1.16.0   // Prometheus metrics
    github.com/pkg/errors v0.9.1                  // error wrapping
)
```  

---

## 3. Configuration (`configs/config.yaml`)
```yaml
# Database connection and environment settings
database:
  dsn: "postgres://<user>:<pass>@<host>:<port>/<db>?sslmode=disable"

# Logging settings
logging:
  level: "info"    # debug | info | warn | error
  kafka_brokers:
    - "broker1:9092"
    - "broker2:9092"
  kafka_topic: "logging"

# Metrics settings
metrics:
  listen_address: ":2117"
  metrics_path: "/metrics"
```  

---

## 4. CLI Commands & Usage  
**Build:**
```bash
go version  # should report go version go1.24.2
go build -o bin/kaeshi-migrate ./cmd/migrate
```

**Usage:**
```bash
kaeshi <command>
```
- `up`         : Apply all pending migrations  
- `down`       : Roll back all applied migrations (to version 0)
- `rollback`   : Roll back exactly one migration step
- `create`     : Generate new up & down migration files
- `status`     : Show current migration version and pending steps
- `version`    : Print current migration version
- `commit`: CHANGE THIS FILE AND REPLACE THIS FILE

Include global panic handler that logs stack traces and sends to Kafka, then exits with proper code.

---

## 5. Config Loader (`internal/config/loader.go`)
- Use **spf13/viper** to load `configs/config.yaml`.  
- Validate required fields: `database.dsn`, `logging.level`, `kafka_brokers`, `kafka_topic`, `metrics.listen_address`.  
- Return typed `Config` struct:
  ```go
  type Config struct {
      Database struct { Dsn string }
      Logging  struct { Level string; KafkaBrokers []string; KafkaTopic string }
      Metrics  struct { ListenAddress string; MetricsPath string }
  }
  ```  

---

## 6. Migration Manager (`internal/migration/manager.go`)
- Wrap **github.com/golang-migrate/migrate/v4**.  
- Methods:
  - `Up() error`
  - `Down() error`
  - `Steps(n int) error`
  - `Version() (uint, bool, error)`
  - `Status() (currentVersion uint, pending int, err error)`
- Use DB advisory lock.  
- Wrap errors with context via **github.com/pkg/errors**.

---

## 7. Metrics Server (`internal/metrics/server.go`)
- Use **prometheus/client_golang** to register:
  - `migrations_applied_total` (Counter)
  - `migration_duration_seconds` (Histogram)
  - `migrations_rollback_total` (Counter)
- Expose HTTP server on `Config.Metrics.ListenAddress` and path `Config.Metrics.MetricsPath`.
- Start server in a separate goroutine from CLI.

---

## 8. Structured Logging to Kafka (`pkg/logger/logger.go`)
- Use **sirupsen/logrus** with JSONFormatter.
- Initialize Kafka producer via **Shopify/sarama**.
- Wrap logger to send each entry to:
  1. Stdout
  2. Kafka topic `Config.Logging.KafkaTopic`
- Fields: `timestamp`, `level`, `component`, `message`, `error.stack`.

---

## 9. CLI Entrypoint (`cmd/migrate/main.go`)
- Use **spf13/cobra** for commands.
- Before parsing commands, install global panic interceptor:
  ```go
  defer func() {
      if r := recover(); r != nil {
          logger.WithField("panic", r).Error("unhandled panic")
          os.Exit(2)
      }
  }()
  ```
- Initialize Config, Logger, Metrics.
- Start Metrics server.
- Execute subcommand: up/down/rollback/status/version.
- Record metrics:
  - Increment counters before/after operations.
  - Observe duration histograms.
- On error, wrap and log full stack, exit with code.

---

## 10. GitLab CI Integration (`.gitlab-ci.yml`)
```yaml
stages:
  - test
  - deploy

variables:
  DATABASE_URL: postgres://ci:ci@postgres:5432/ci_db?sslmode=disable

services:
  - name: postgres:15
    alias: postgres
    command: ["--health-check=pg_isready -U ci -d ci_db"]

before_script:
  - go version
  - go mod download

test_migrations:
  stage: test
  script:
    - go run ./cmd/migrate up
    - go test ./...
    - go run ./cmd/migrate down
  tags:
    - docker

deploy_migrations:
  stage: deploy
  script:
    - go run ./cmd/migrate up
  when: manual
  tags:
    - docker
```  

---

## 11. Best Practices & Production Hardening  
- **Locking**: advisory lock in DB.  
- **Idempotency**: safe re-runs.  
- **Audit**: log migration version, user, timestamp.  
- **Monitoring**: Prometheus metrics at `:2117/metrics`.  
- **Secrets**: use GitLab CI/CD variables for DSN and Kafka brokers.  
- **Documentation**: update README.md.  
- **Access Control**: restrict `deploy_migrations` to authorized roles.  

---

**References**
- [golang-migrate/migrate](https://github.com/golang-migrate/migrate)
- [spf13/viper](https://github.com/spf13/viper)
- [sirupsen/logrus](https://github.com/sirupsen/logrus)
- [spf13/cobra](https://github.com/spf13/cobra)
- [Shopify/sarama](https://github.com/Shopify/sarama)
- [prometheus/client_golang](https://github.com/prometheus/client_golang)
- [pkg/errors](https://github.com/pkg/errors)
