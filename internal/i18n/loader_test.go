package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_GetTranslationMap_LoadsFromDisk(t *testing.T) {
	base := t.TempDir()
	uiDir := filepath.Join(base, "lua/shared/Translate/EN")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Loader 会加载 Sandbox/UI/Tooltip 三类文件；这里我们只放 UI_EN.txt。
	uiPath := filepath.Join(uiDir, "UI_EN.txt")
	content := `{
UI_ServerOption_TestKey_tooltip = "Hello Tooltip",
Sandbox_TestKey = "Hello Label",
}`
	if err := os.WriteFile(uiPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	loader := NewLoader(base)
	dict := loader.GetTranslationMap("EN")

	if got := dict["UI_ServerOption_TestKey_tooltip"]; got != "Hello Tooltip" {
		t.Fatalf("unexpected tooltip: %q", got)
	}
}

func TestLoader_GetTranslationMap_LoadsBuild42JSON(t *testing.T) {
	base := t.TempDir()
	uiDir := filepath.Join(base, "lua/shared/Translate/CN")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	uiPath := filepath.Join(uiDir, "UI.json")
	content := `{"UI_ServerOption_TestKey_tooltip":"JSON Tooltip","UI_ServerOption_TestKey":"JSON Label"}`
	if err := os.WriteFile(uiPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write ui: %v", err)
	}

	loader := NewLoader(base)
	dict := loader.GetTranslationMap("CN")

	if got := dict["UI_ServerOption_TestKey_tooltip"]; got != "JSON Tooltip" {
		t.Fatalf("unexpected JSON tooltip: %q", got)
	}
	if got := dict["UI_ServerOption_TestKey"]; got != "JSON Label" {
		t.Fatalf("unexpected JSON label: %q", got)
	}
}

func TestTranslateKey_LabelAndTooltip(t *testing.T) {
	dict := TranslationMap{
		"UI_ServerOption_Foo":         "Foo Label",
		"UI_ServerOption_Foo_tooltip": "Foo Tip",
	}

	label, tip := TranslateKey(dict, "Foo", "UI_ServerOption_")
	if label != "Foo Label" {
		t.Fatalf("label=%q", label)
	}
	if tip != "Foo Tip" {
		t.Fatalf("tip=%q", tip)
	}
}

func TestNormalizeTranslationText(t *testing.T) {
	got := normalizeTranslationText(`第一行\n第二行<br>第三行\t缩进`)
	want := "第一行\n第二行\n第三行\t缩进"
	if got != want {
		t.Fatalf("normalizeTranslationText()=%q, want %q", got, want)
	}
}
