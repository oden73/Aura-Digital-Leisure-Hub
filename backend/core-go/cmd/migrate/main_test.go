package main

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadMigrations_SortsLexicallyAndSkipsNonSQL(t *testing.T) {
	dir := t.TempDir()

	files := []string{
		"0002_indexes.sql",
		"0001_init.sql",
		"README.md",
		"0010_later.sql",
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("-- noop"), 0o600); err != nil {
			t.Fatalf("seed %s: %v", f, err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	got, err := loadMigrations(dir)
	if err != nil {
		t.Fatalf("loadMigrations: %v", err)
	}
	want := []string{"0001_init", "0002_indexes", "0010_later"}
	gotVersions := make([]string, len(got))
	for i, m := range got {
		gotVersions[i] = m.version
	}
	if !reflect.DeepEqual(gotVersions, want) {
		t.Fatalf("versions mismatch: got %v want %v", gotVersions, want)
	}
}

func TestLoadMigrations_EmptyDirIsError(t *testing.T) {
	if _, err := loadMigrations(t.TempDir()); err == nil {
		t.Fatal("expected error for empty migrations dir")
	}
}

func TestPrintStatus_Snapshot(t *testing.T) {
	migs := []migration{{version: "0001_init"}, {version: "0002_indexes"}}
	applied := map[string]struct{}{"0001_init": {}}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	stdout := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = stdout })

	printStatus(migs, applied)
	_ = w.Close()

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	out := string(buf[:n])
	if want := "APPLIED   0001_init"; !strings.Contains(out, want) {
		t.Fatalf("missing %q in output:\n%s", want, out)
	}
	if want := "PENDING   0002_indexes"; !strings.Contains(out, want) {
		t.Fatalf("missing %q in output:\n%s", want, out)
	}
}

func TestDefaultMigrationsDir_WhenRelativePathExists(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	base := t.TempDir()
	rel := filepath.Join("backend", "db", "migrations")
	if err := os.MkdirAll(filepath.Join(base, rel), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, rel, "0001_x.sql"), []byte("-- x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(base); err != nil {
		t.Fatal(err)
	}

	got := defaultMigrationsDir()
	if got != "backend/db/migrations" {
		t.Fatalf("unexpected dir: %q", got)
	}
}

func TestPrintVersion_NoneApplied(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	stdout := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = stdout })

	printVersion([]migration{{version: "0001_x"}}, map[string]struct{}{})
	_ = w.Close()
	var buf [64]byte
	n, _ := r.Read(buf[:])
	out := string(buf[:n])
	if !strings.Contains(out, "(none)") {
		t.Fatalf("expected (none), got %q", out)
	}
}

func TestPrintVersion_LatestLexical(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	stdout := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = stdout })

	migs := []migration{{version: "0001_a"}, {version: "0002_b"}}
	applied := map[string]struct{}{"0001_a": {}, "0002_b": {}}
	printVersion(migs, applied)
	_ = w.Close()
	var buf [64]byte
	n, _ := r.Read(buf[:])
	out := strings.TrimSpace(string(buf[:n]))
	if out != "0002_b" {
		t.Fatalf("want 0002_b, got %q", out)
	}
}

func TestUsage_WritesHelpToStderr(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	stderr := os.Stderr
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = stderr })

	usage()
	_ = w.Close()

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "Usage:") || !strings.Contains(s, "up|status|version") {
		t.Fatalf("unexpected usage output: %q", s)
	}
}
