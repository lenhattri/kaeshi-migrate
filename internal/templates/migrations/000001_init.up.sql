-- Example initial migration
CREATE TABLE IF NOT EXISTS schema_migrations (
    id SERIAL PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version VARCHAR(50) NOT NULL UNIQUE,
    dirty BOOLEAN NOT NULL DEFAULT FALSE
);

-- Track who executed migrations and when
CREATE TABLE IF NOT EXISTS migrations_history (
    id SERIAL PRIMARY KEY,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    action VARCHAR(20) NOT NULL,
    version VARCHAR(50) NOT NULL,
    executed_by VARCHAR(100) NOT NULL,
    committed BOOLEAN NOT NULL DEFAULT FALSE,
    sha256 TEXT NOT NULL,
    
);


