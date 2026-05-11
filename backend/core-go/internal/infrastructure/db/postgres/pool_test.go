package postgres

import (
	"context"
	"testing"
	"time"
)

func TestConnect_InvalidDSN(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := Connect(ctx, "not-a-valid-postgres-dsn")
	if err == nil {
		t.Fatal("expected error for invalid dsn")
	}
}

func TestConnect_RefusedOrUnreachable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Valid URL shape but nothing listening on this port — Ping should fail.
	_, err := Connect(ctx, "postgres://user:pass@127.0.0.1:65534/postgres?connect_timeout=1")
	if err == nil {
		t.Fatal("expected error when database is unreachable")
	}
}
