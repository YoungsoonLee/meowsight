// Package migrations holds embedded SQL migration files for PostgreSQL and ClickHouse.
package migrations

import "embed"

// PostgresFS contains all PostgreSQL migration SQL files.
//
//go:embed postgres/*.sql
var PostgresFS embed.FS

// ClickHouseFS contains all ClickHouse migration SQL files.
//
//go:embed clickhouse/*.sql
var ClickHouseFS embed.FS
