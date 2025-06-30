package templates

import _ "embed"

//go:embed default_config.yml
var DefaultConfig string

//go:embed migrations/000001_init.up.sql
var InitUp string

//go:embed migrations/000001_init.down.sql
var InitDown string
