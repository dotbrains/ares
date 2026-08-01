package hostfs

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type FS struct {
	Root string
	Now  time.Time
}

func (fs FS) Path(path string) string {
	if fs.Root == "" {
		return path
	}
	return filepath.Join(fs.Root, strings.TrimPrefix(path, "/"))
}

func (fs FS) DisplayPath(path string) string {
	return strings.TrimPrefix(path, fs.Root)
}

func (fs FS) WriteFile(path string, data []byte, perm os.FileMode) error {
	fullPath := fs.Path(path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, data, perm)
}

func (fs FS) Remove(path string) error {
	return os.Remove(fs.Path(path))
}

func (fs FS) Backup(path string) (string, bool, error) {
	source := fs.Path(path)
	if _, err := os.Stat(source); err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	backupPath := source + ".ares." + fs.Now.Format("20060102-150405") + ".bak"
	if _, err := os.Stat(backupPath); err == nil {
		return fs.DisplayPath(backupPath), false, nil
	} else if !os.IsNotExist(err) {
		return "", false, err
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return "", false, err
	}
	if err := os.WriteFile(backupPath, data, 0o600); err != nil {
		return "", false, err
	}
	return fs.DisplayPath(backupPath), true, nil
}

func (fs FS) RestoreNewestBackup(path string) (string, bool, error) {
	fullPath := fs.Path(path)
	matches, err := filepath.Glob(fullPath + ".ares.*.bak")
	if err != nil || len(matches) == 0 {
		return "", false, err
	}
	sort.Strings(matches)
	backup := matches[len(matches)-1]
	data, err := os.ReadFile(backup)
	if err != nil {
		return "", true, err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return "", true, err
	}
	if err := os.WriteFile(fullPath, data, 0o600); err != nil {
		return "", true, err
	}
	return fs.DisplayPath(backup), true, nil
}
