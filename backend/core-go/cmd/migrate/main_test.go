package main

import (
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
