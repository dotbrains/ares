package hostfs

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBackupAndRestoreNewestBackupUseRoot(t *testing.T) {
	root := t.TempDir()
	fs := FS{Root: root, Now: time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)}
	if err := fs.WriteFile("/etc/example.conf", []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	backup, created, err := fs.Backup("/etc/example.conf")
	if err != nil {
		t.Fatal(err)
	}
	if !created || backup != "/etc/example.conf.ares.20260801-120000.bak" {
		t.Fatalf("backup = %q/%t", backup, created)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "example.conf"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	restoredFrom, restored, err := fs.RestoreNewestBackup("/etc/example.conf")
	if err != nil {
		t.Fatal(err)
	}
	if !restored || restoredFrom != backup {
		t.Fatalf("restored = %q/%t", restoredFrom, restored)
	}
	data, err := os.ReadFile(filepath.Join(root, "etc", "example.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original" {
		t.Fatalf("restored data = %q", data)
	}
}

func TestWriteFileReplacesAtomically(t *testing.T) {
	root := t.TempDir()
	fs := FS{Root: root}
	if err := fs.WriteFile("/etc/example.conf", []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := fs.WriteFile("/etc/example.conf", []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "etc", "example.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new\n" {
		t.Fatalf("data = %q", data)
	}
	info, err := os.Stat(filepath.Join(root, "etc", "example.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("mode = %v, want 0644", got)
	}
	matches, err := filepath.Glob(filepath.Join(root, "etc", ".example.conf.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary host files left behind: %v", matches)
	}
}
