package atomicfile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteReplacesFileAndCleansTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "managed.conf")
	if err := os.WriteFile(path, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Write(path, []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new\n" {
		t.Fatalf("data = %q", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("mode = %v, want 0644", got)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".managed.conf.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files left behind: %v", matches)
	}
}

func TestWriteReturnsErrorForMissingParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "managed.conf")
	err := Write(path, []byte("new\n"), 0o644)
	if err == nil {
		t.Fatal("expected missing parent directory error")
	}
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if !os.IsNotExist(err) {
		t.Fatalf("err = %v, want missing parent directory", err)
	}
}
