//go:build integration

// Integration tests require a running Postgres reachable via
// $INTEGRATION_DATABASE_URL. Run with:
//
//	INTEGRATION_DATABASE_URL=postgres://aura:aura@localhost:5432/aura_test \
//	  go test -tags=integration ./internal/infrastructure/repository/postgres/...
//
// In CI the workflow spins up an ephemeral postgres service and runs
// the migrations from backend/db/migrations against it before invoking
// the integration suite.

package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"aura/backend/core-go/internal/domain/entities"
	dbpostgres "aura/backend/core-go/internal/infrastructure/db/postgres"
)

var (
	sharedPool   *dbpostgres.Pool
	sharedPoolMu sync.Mutex
)

// mustGetPool returns a process-wide pgxpool wired to the integration
// database. The first caller pays the cost of running migrations; every
// subsequent caller gets a freshly-truncated database so tests do not
// see each other's data.
func mustGetPool(t *testing.T) *dbpostgres.Pool {
	t.Helper()

	dsn := os.Getenv("INTEGRATION_DATABASE_URL")
	if dsn == "" {
		t.Skip("INTEGRATION_DATABASE_URL not set; skipping postgres integration tests")
	}

	sharedPoolMu.Lock()
	defer sharedPoolMu.Unlock()

	if sharedPool == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		p, err := dbpostgres.Connect(ctx, dsn)
		if err != nil {
			t.Fatalf("connect to %s: %v", dsn, err)
		}
		if err := applyMigrations(ctx, p); err != nil {
			t.Fatalf("apply migrations: %v", err)
		}
		sharedPool = p
	}

	resetTables(t, sharedPool)
	return sharedPool
}

// applyMigrations runs every *.sql in backend/db/migrations in lexical
// order. Each file owns its own BEGIN/COMMIT (matching cmd/migrate's
// convention) so we just hand the body to the pool.
func applyMigrations(ctx context.Context, p *dbpostgres.Pool) error {
	dir := findMigrationsDir()
	if dir == "" {
		return errors.New("migrations directory not found")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	for _, f := range files {
		body, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := p.Exec(ctx, string(body)); err != nil {
			// Most types/extensions are guarded with IF NOT EXISTS;
			// for the rest we tolerate "already exists" so re-runs
			// against a populated database keep working.
			msg := err.Error()
			if strings.Contains(msg, "already exists") {
				continue
			}
			return fmt.Errorf("apply %s: %w", f, err)
		}
	}
	return nil
}

// findMigrationsDir locates backend/db/migrations starting from the
// test binary's working directory and walking upward. The walk stops as
// soon as a `db/migrations` sibling is found, so the same setup works
// from this package, from the module root and from a CI checkout where
// extra path components may be present.
func findMigrationsDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := cwd
	for {
		candidate := filepath.Join(dir, "db", "migrations")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// resetTables wipes every table that integration tests touch in one
// statement so per-test ordering does not matter. RESTART IDENTITY
// keeps BIGSERIAL counters predictable across runs.
func resetTables(t *testing.T, p *dbpostgres.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := p.Exec(ctx, `
		TRUNCATE
			external_accounts,
			user_interactions,
			vector_store,
			book_details,
			cinema_details,
			game_details,
			base_items,
			users
		RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// ---------- shared fixtures -------------------------------------------------

// mustCreateUser inserts a user row and returns its UUID. PasswordHash
// is intentionally a fixed sentinel — the schema only requires a NOT
// NULL value, no test in this package re-validates the hash.
func mustCreateUser(t *testing.T, repo *UserRepo, username, email string) string {
	t.Helper()
	u, err := repo.Create(entities.User{
		Username:     username,
		Email:        email,
		PasswordHash: "deadbeef$cafebabe",
	})
	if err != nil {
		t.Fatalf("create user (%s/%s): %v", username, email, err)
	}
	return u.ID
}

// mustSaveItem persists a base_items row (no media-specific details
// unless the caller pre-populated them) and returns the assigned id.
// SaveItem writes the generated UUID back into the supplied Item so
// callers do not need a follow-up SELECT.
func mustSaveItem(t *testing.T, repo *MetadataRepo, item entities.Item) string {
	t.Helper()
	if err := repo.SaveItem(&item); err != nil {
		t.Fatalf("save item %q: %v", item.Title, err)
	}
	if item.ID == "" {
		t.Fatalf("save item %q: ID was not populated", item.Title)
	}
	return item.ID
}

// mustSaveInteraction inserts a Rui row.
func mustSaveInteraction(
	t *testing.T,
	repo *InteractionRepo,
	userID, itemID string,
	status entities.InteractionStatus,
	rating int,
) {
	t.Helper()
	now := time.Now().UTC()
	err := repo.Save(entities.Interaction{
		UserID:    userID,
		ItemID:    itemID,
		Status:    status,
		Rating:    rating,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("save interaction (%s,%s,%d): %v", userID, itemID, rating, err)
	}
}
