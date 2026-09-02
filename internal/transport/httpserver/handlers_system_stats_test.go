package httpserver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadGameVersion(t *testing.T) {
	root := t.TempDir()
	logs := filepath.Join(root, "Logs")
	if err := os.MkdirAll(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(logs, "server.txt")
	if err := os.WriteFile(path, []byte("old\nversion=42.20.4 b0bbce05d5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readGameVersion(root, ""); got != "42.20.4" {
		t.Fatalf("readGameVersion()=%q, want 42.20.4", got)
	}
}

func TestReadGameVersionUnknown(t *testing.T) {
	if got := readGameVersion(t.TempDir(), ""); got != "unknown" {
		t.Fatalf("readGameVersion()=%q, want unknown", got)
	}
}
