package apply

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasAuthorizedKeysRequiresNonEmptyAuthorizedKeys(t *testing.T) {
	root := t.TempDir()
	path := "/home/ares/.ssh/authorized_keys"
	fullPath := filepath.Join(root, "home", "ares", ".ssh")
	if err := os.MkdirAll(fullPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fullPath, "authorized_keys"), []byte("\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if hasAuthorizedKeys(root, []string{path}) {
		t.Fatal("empty authorized_keys should not satisfy SSH hardening guard")
	}
	if err := os.WriteFile(filepath.Join(fullPath, "authorized_keys"), []byte("ssh-ed25519 AAAA test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !hasAuthorizedKeys(root, []string{path}) {
		t.Fatal("non-empty authorized_keys should satisfy SSH hardening guard")
	}
}
