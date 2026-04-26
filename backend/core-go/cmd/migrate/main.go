// Command migrate is the database migration runner for the Aura Go core.
//
// Convention: each *.sql file in the migrations directory is a complete
// migration that manages its own transaction (BEGIN/COMMIT inside the
// file). The runner does NOT wrap files in an additional transaction —
// some statements (CREATE INDEX CONCURRENTLY, CREATE TYPE) cannot run
// inside a wrapping BEGIN that is later wrapped again. The runner
// records successful application in a small bookkeeping table
// (schema_migrations) so subsequent runs are idempotent.
//
// Usage:
//
//	migrate up                 # apply every pending migration
//	migrate status             # list applied / pending migrations
//	migrate version            # print the latest applied version
//
// Connection is taken from the DATABASE_URL env var (same as the API),
// or from --dsn. The migrations directory defaults to backend/db/migrations
// relative to the binary; --dir overrides.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const schemaMigrationsDDL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version    TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`

type migration struct {
	version string
	path    string
}

func main() {
	var (
		dsn  = flag.String("dsn", os.Getenv("DATABASE_URL"), "Postgres connection string (defaults to $DATABASE_URL)")
		dir  = flag.String("dir", defaultMigrationsDir(), "Directory containing *.sql migration files")
		help = flag.Bool("help", false, "Show usage")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if *help || flag.NArg() == 0 {
		usage()
		return
	}
	if *dsn == "" {
		logger.Error("missing dsn", "hint", "set DATABASE_URL or pass --dsn")
		os.Exit(2)
	}

	cmd := flag.Arg(0)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	conn, err := pgx.Connect(ctx, *dsn)
	if err != nil {
		logger.Error("connect_failed", "error", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	if _, err := conn.Exec(ctx, schemaMigrationsDDL); err != nil {
		logger.Error("bootstrap_table_failed", "error", err)
		os.Exit(1)
	}

	migrations, err := loadMigrations(*dir)
	if err != nil {
		logger.Error("load_migrations_failed", "dir", *dir, "error", err)
		os.Exit(1)
	}

	applied, err := loadApplied(ctx, conn)
	if err != nil {
		logger.Error("load_applied_failed", "error", err)
		os.Exit(1)
	}

	switch cmd {
	case "up":
		if err := runUp(ctx, conn, migrations, applied, logger); err != nil {
			logger.Error("migrate_up_failed", "error", err)
			os.Exit(1)
		}
	case "status":
		printStatus(migrations, applied)
	case "version":
		printVersion(migrations, applied)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: migrate [--dsn=URL] [--dir=PATH] <up|status|version>")
	flag.PrintDefaults()
}

// defaultMigrationsDir returns "backend/db/migrations" resolved relative
// to the executable when possible, falling back to a path that matches
// running the binary from the repo root.
func defaultMigrationsDir() string {
	const rel = "backend/db/migrations"
	if _, err := os.Stat(rel); err == nil {
		return rel
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), rel)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return rel
}

func loadMigrations(dir string) ([]migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}
	out := make([]migration, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		out = append(out, migration{
			version: strings.TrimSuffix(name, ".sql"),
			path:    filepath.Join(dir, name),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no *.sql files in %s", dir)
	}
	// Lexical sort works because the project convention prefixes files
	// with a zero-padded numeric version (0001_, 0002_, ...).
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

func loadApplied(ctx context.Context, conn *pgx.Conn) (map[string]struct{}, error) {
	rows, err := conn.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]struct{}{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = struct{}{}
	}
	return out, rows.Err()
}

func runUp(
	ctx context.Context,
	conn *pgx.Conn,
	migrations []migration,
	applied map[string]struct{},
	logger *slog.Logger,
) error {
	pending := 0
	for _, m := range migrations {
		if _, ok := applied[m.version]; ok {
			continue
		}
		pending++
		body, err := os.ReadFile(m.path)
		if err != nil {
			return fmt.Errorf("read %s: %w", m.path, err)
		}

		logger.Info("apply", "version", m.version)
		// The migration file owns its transaction, so we execute the
		// SQL as-is; any failure inside leaves the database in the
		// state defined by the file's own ROLLBACK semantics.
		if _, err := conn.Exec(ctx, string(body)); err != nil {
			return fmt.Errorf("apply %s: %w", m.version, err)
		}
		// Record success in our own short transaction so a crash between
		// COMMIT-ing the migration and INSERT-ing the row at most
		// re-runs an already-applied (idempotent by convention) file.
		if _, err := conn.Exec(
			ctx,
			`INSERT INTO schema_migrations(version) VALUES ($1)
             ON CONFLICT (version) DO NOTHING`,
			m.version,
		); err != nil {
			return fmt.Errorf("record %s: %w", m.version, err)
		}
	}
	if pending == 0 {
		logger.Info("nothing_to_apply")
	} else {
		logger.Info("done", "applied", pending)
	}
	return nil
}

func printStatus(migrations []migration, applied map[string]struct{}) {
	for _, m := range migrations {
		mark := "PENDING"
		if _, ok := applied[m.version]; ok {
			mark = "APPLIED"
		}
		fmt.Printf("%-8s  %s\n", mark, m.version)
	}
}

func printVersion(migrations []migration, applied map[string]struct{}) {
	var latest string
	for _, m := range migrations {
		if _, ok := applied[m.version]; ok {
			latest = m.version
		}
	}
	if latest == "" {
		fmt.Println("(none)")
		return
	}
	fmt.Println(latest)
}
