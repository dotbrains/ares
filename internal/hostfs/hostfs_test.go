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
