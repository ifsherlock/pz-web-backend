package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateListRestoreDelete(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Server"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "Saves", "Multiplayer"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Server", "servertest.ini"), []byte("Password=one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Saves", "Multiplayer", "world.bin"), []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewService(root)
	panelDir := filepath.Join(root, "panel")
	if err := os.MkdirAll(panelDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(panelDir, "panel_settings.json"), []byte(`{"game_branch":"public"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	svc.PanelDir = panelDir
	record, err := svc.Create("nightly test", "42.20.4", 3)
	if err != nil {
		t.Fatal(err)
	}
	if record.SizeBytes == 0 || record.GameVersion != "42.20.4" {
		t.Fatalf("unexpected record: %+v", record)
	}
	items, err := svc.List()
	if err != nil || len(items) != 1 {
		t.Fatalf("List() items=%d err=%v", len(items), err)
	}
	if err := os.WriteFile(filepath.Join(root, "Server", "servertest.ini"), []byte("Password=changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := svc.Restore(record.ID); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "Server", "servertest.ini"))
	if err != nil || string(data) != "Password=one\n" {
		t.Fatalf("restored data=%q err=%v", data, err)
	}
	if err := svc.Delete(record.ID); err != nil {
		t.Fatal(err)
	}
	items, err = svc.List()
	if err != nil || len(items) != 0 {
		t.Fatalf("after delete items=%d err=%v", len(items), err)
	}
}

func TestBackupExcludesInstallDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Server"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "game"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Server", "server.ini"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "game", "binary"), []byte("must not be archived"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewService(root)
	record, err := svc.Create("", "42.20.4", 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Restore(record.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "game", "binary")); err != nil {
		t.Fatal("source game file should remain", err)
	}
}
