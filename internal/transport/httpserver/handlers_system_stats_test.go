package httpserver

import (
	"os"
	"path/filepath"
	"testing"

	"pz-web-backend/internal/config"
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

func TestResolveGameQueryPort(t *testing.T) {
	items := []config.Item{{Key: "UDPPort", Value: "16262"}, {Key: "DefaultPort", Value: "16270"}}
	if got := resolveGameQueryPort(items, nil); got != 16270 {
		t.Fatalf("resolveGameQueryPort()=%d, want 16270", got)
	}
}

func TestResolveGameQueryPortFallsBackForInvalidConfig(t *testing.T) {
	items := []config.Item{{Key: "DefaultPort", Value: "invalid"}}
	if got := resolveGameQueryPort(items, nil); got != 16261 {
		t.Fatalf("resolveGameQueryPort()=%d, want 16261", got)
	}
}
