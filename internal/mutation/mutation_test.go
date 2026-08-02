package mutation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOperatorDryRunCommandAndFileMutation(t *testing.T) {
	root := t.TempDir()
	operator := Operator{Root: root}

	runResult, err := operator.Run("systemctl", "reload", "ssh")
	if err != nil {
		t.Fatal(err)
	}
	if len(runResult.Applied) != 1 || runResult.Applied[0] != "would run: systemctl reload ssh" {
		t.Fatalf("run result = %+v", runResult)
	}

	path := filepath.Join(root, "managed.conf")
	if err := os.WriteFile(path, []byte("managed"), 0o644); err != nil {
		t.Fatal(err)
	}
	removeResult := operator.Remove("/managed.conf")
	if len(removeResult.Applied) != 1 || removeResult.Applied[0] != "removed /managed.conf" {
		t.Fatalf("remove result = %+v", removeResult)
	}
}

func TestOperatorWriteFileCreatesParentDirectory(t *testing.T) {
	root := t.TempDir()
	operator := Operator{Root: root}
	result, err := operator.WriteFile("/etc/ares/example.conf", []byte("ok\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Applied) != 1 || result.Applied[0] != "wrote /etc/ares/example.conf" {
		t.Fatalf("write result = %+v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "etc", "ares", "example.conf")); err != nil {
		t.Fatalf("missing written file: %v", err)
	}
}
