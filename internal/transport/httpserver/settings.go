package httpserver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"pz-web-backend/internal/infra/fs"
)

// PanelSettings 面板持久化设置。
// 保存在挂载卷(容器内 /opt/pz-web-backend/panel_settings.json)，
// 这样不重建镜像也能修改配置，start-pz.sh 也可读取内存设置。
type PanelSettings struct {
	SteamAPIKey string `json:"steam_api_key"`
	MemoryLimit string `json:"memory_limit"` // 例: 3g
}

const panelSettingsFilename = "panel_settings.json"

var memoryLimitPattern = regexp.MustCompile(`^[1-9][0-9]*[mMgG]$`)

func normalizeMemoryLimit(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if !memoryLimitPattern.MatchString(value) {
		return "", fmt.Errorf("memory limit must use a positive integer followed by m or g, for example 3072m or 4g")
	}
	return strings.ToLower(value), nil
}

// settingsStore 负责读写面板设置文件。
type settingsStore struct {
	fs   fs.FS
	path string
}

func newSettingsStore(fsys fs.FS, installDir string) *settingsStore {
	// 与 WorkshopCache 同目录，均在挂载卷 /opt/pz-web-backend 下
	path := filepath.Join(installDir, "..", "pz-web-backend", panelSettingsFilename)
	return &settingsStore{fs: fsys, path: path}
}

// Load 读取设置；文件不存在时返回空设置(不报错)。
func (s *settingsStore) Load() PanelSettings {
	var settings PanelSettings
	if s.fs == nil {
		return settings
	}
	if data, err := os.ReadFile(s.path); err == nil {
		_ = json.Unmarshal(data, &settings)
	}
	return settings
}

// Save 写入设置；确保父目录存在。
func (s *settingsStore) Save(settings PanelSettings) error {
	if s.fs == nil {
		return nil
	}
	if err := s.fs.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return s.fs.WriteFile(s.path, data, 0o644)
}
