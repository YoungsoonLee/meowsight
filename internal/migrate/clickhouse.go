package migrate

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"

	"github.com/YoungsoonLee/meowsight/migrations"
)

// RunClickHouse applies all unapplied ClickHouse .up.sql migrations from the
// embedded migrations/clickhouse directory. Applied migrations are tracked in
// the schema_migrations table.
func RunClickHouse(ctx context.Context, host string, port int, database, user, password string) error {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", host, port)},
		Auth: clickhouse.Auth{
			Database: database,
			Username: user,
			Password: password,
		},
	})
	if err != nil {
		return fmt.Errorf("clickhouse open: %w", err)
	}
	defer conn.Close()

	if err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version String,
			applied_at DateTime DEFAULT now()
		) ENGINE = MergeTree() ORDER BY version
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	files, err := listUpMigrations(migrations.ClickHouseFS, "clickhouse")
	if err != nil {
		return err
	}

	for _, name := range files {
		var count uint64
		if err := conn.QueryRow(ctx,
			`SELECT count() FROM schema_migrations WHERE version = ?`, name,
		).Scan(&count); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if count > 0 {
			continue
		}

		sqlBytes, err := migrations.ClickHouseFS.ReadFile("clickhouse/" + name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		slog.Info("applying clickhouse migration", "version", name)
		// ClickHouse driver runs one statement per Exec; split on `;`.
		for _, stmt := range splitSQL(string(sqlBytes)) {
			if err := conn.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("apply %s: %w", name, err)
			}
		}
		if err := conn.Exec(ctx,
			`INSERT INTO schema_migrations (version) VALUES (?)`, name,
		); err != nil {
			return fmt.Errorf("record %s: %w", name, err)
		}
	}

	return nil
}

// splitSQL splits a SQL file into individual statements separated by `;`,
// stripping line comments and empty statements.
func splitSQL(s string) []string {
	var out []string
	for raw := range strings.SplitSeq(s, ";") {
		var b strings.Builder
		for line := range strings.SplitSeq(raw, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "--") {
				continue
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
		stmt := strings.TrimSpace(b.String())
		if stmt != "" {
			out = append(out, stmt)
		}
	}
	return out
}
