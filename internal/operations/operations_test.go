package operations

import "testing"

func TestSummaryRendersTypedOperations(t *testing.T) {
	summary := Summary([]Operation{
		{Kind: WriteFile, Path: "/etc/example.conf"},
		{Kind: BackupFile, Path: "/etc/example.conf"},
		{Kind: RunCommand, Command: "systemctl", Args: []string{"reload", "ssh"}},
		{Kind: RollbackNote, Note: "restore /etc/example.conf backup"},
		{Kind: CustomCommand, Plugin: "custom", Phase: "apply", Command: "custom apply"},
	})
	if summary.Files[0] != "/etc/example.conf" {
		t.Fatalf("files = %+v", summary.Files)
	}
	if summary.Backups[0] != "/etc/example.conf" {
		t.Fatalf("backups = %+v", summary.Backups)
	}
	if summary.Commands[0] != "systemctl reload ssh" || summary.Commands[1] != "custom custom apply: custom apply" {
		t.Fatalf("commands = %+v", summary.Commands)
	}
	if summary.RollbackSteps[0] != "restore /etc/example.conf backup" {
		t.Fatalf("rollback = %+v", summary.RollbackSteps)
	}
}

func TestRollbackPreviewSkipsManagedFileWithBackup(t *testing.T) {
	preview := RollbackPreview(Summary([]Operation{
		{Kind: WriteFile, Path: "/etc/example.conf"},
		{Kind: BackupFile, Path: "/etc/example.conf"},
		{Kind: WriteFile, Path: "/etc/managed.conf"},
	}))
	if len(preview) != 2 {
		t.Fatalf("preview = %+v", preview)
	}
	if preview[0] != "would restore newest backup for /etc/example.conf" || preview[1] != "would remove /etc/managed.conf" {
		t.Fatalf("preview = %+v", preview)
	}
}
