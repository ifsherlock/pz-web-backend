package httpserver

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"pz-web-backend/internal/application/configapp"
	"pz-web-backend/internal/backup"
	"pz-web-backend/internal/config"
)

var nativeBackupKeys = map[string]bool{
	"SaveWorldEveryMinutes":  true,
	"BackupsCount":           true,
	"BackupsOnStart":         true,
	"BackupsOnVersionChange": true,
	"BackupsPeriod":          true,
}

type backupSettingsResponse struct {
	Enabled        bool            `json:"enabled"`
	IntervalHours  int             `json:"interval_hours"`
	MaxVersions    int             `json:"max_versions"`
	ServerSettings []config.Item   `json:"server_settings"`
	Backups        []backup.Record `json:"backups"`
}

func (a App) handleGetBackupSettings(c *gin.Context) {
	settings := a.backupSettings()
	items, err := a.ConfigApp.GetServerConfig(strings.ToUpper(c.DefaultQuery("lang", "CN")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	native := make([]config.Item, 0, len(nativeBackupKeys))
	for _, item := range items {
		if nativeBackupKeys[item.Key] {
			native = append(native, item)
		}
	}
	records, err := a.Backups.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, backupSettingsResponse{Enabled: settings.Enabled, IntervalHours: settings.IntervalHours, MaxVersions: settings.MaxVersions, ServerSettings: native, Backups: records})
}

func (a App) handleSaveBackupSettings(c *gin.Context) {
	var req struct {
		Enabled        *bool         `json:"enabled"`
		IntervalHours  *int          `json:"interval_hours"`
		MaxVersions    *int          `json:"max_versions"`
		ServerSettings []config.Item `json:"server_settings"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	settings := a.backupSettings()
	if req.Enabled != nil {
		settings.Enabled = *req.Enabled
	}
	if req.IntervalHours != nil {
		if *req.IntervalHours < 1 || *req.IntervalHours > 168 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "backup interval must be between 1 and 168 hours"})
			return
		}
		settings.IntervalHours = *req.IntervalHours
	}
	if req.MaxVersions != nil {
		if *req.MaxVersions < 1 || *req.MaxVersions > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "backup retention must be between 1 and 100 versions"})
			return
		}
		settings.MaxVersions = *req.MaxVersions
	}
	if len(req.ServerSettings) > 0 {
		items, err := a.ConfigApp.GetServerConfig("CN")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		updates := map[string]string{}
		for _, item := range req.ServerSettings {
			if nativeBackupKeys[item.Key] {
				updates[item.Key] = item.Value
			}
		}
		for i := range items {
			if value, ok := updates[items[i].Key]; ok {
				items[i].Value = value
			}
		}
		if err := a.ConfigApp.Save(configapp.KindServer, items, false); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if err := a.Settings.Save(settings.toPanelSettings(a.Settings.Load())); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "saved", "enabled": settings.Enabled, "interval_hours": settings.IntervalHours, "max_versions": settings.MaxVersions})
}

func (a App) handleCreateBackup(c *gin.Context) {
	var req struct {
		Note string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	settings := a.backupSettings()
	resume, err := a.stopGameForOperation()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	record, err := a.Backups.Create(req.Note, readGameVersion(a.BaseDataDir, a.LogPath), settings.MaxVersions)
	if resume != nil {
		if startErr := resume(); err == nil {
			err = startErr
		}
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, record)
}

func (a App) handleDeleteBackup(c *gin.Context) {
	if err := a.Backups.Delete(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (a App) handleRestoreBackup(c *gin.Context) {
	resume, err := a.stopGameForOperation()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := a.Backups.Restore(c.Param("id")); err != nil {
		if resume != nil {
			_ = resume()
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if a.ConfigApp.Runner != nil {
		if _, err := a.ConfigApp.Runner.CombinedOutput("chown", "-R", "steam:steam", a.BaseDataDir); err != nil {
			if resume != nil {
				_ = resume()
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fix restored file ownership: " + err.Error()})
			return
		}
		if a.Backups.PanelDir != "" {
			if _, err := a.ConfigApp.Runner.CombinedOutput("chown", "-R", "steam:steam", a.Backups.PanelDir); err != nil {
				if resume != nil {
					_ = resume()
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fix panel settings ownership: " + err.Error()})
				return
			}
		}
	}
	if resume != nil {
		if err := resume(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "restored_and_restarting"})
}

// stopGameForOperation pauses the game for filesystem-consistent backup work.
// It returns a resume callback only when a supervisor runner is configured.
func (a App) stopGameForOperation() (func() error, error) {
	if a.ConfigApp.DevMode || a.ConfigApp.Runner == nil {
		return nil, nil
	}
	output, err := a.ConfigApp.Runner.CombinedOutput("/usr/bin/supervisorctl", "-c", "/etc/supervisor/conf.d/supervisord.conf", "stop", "pzserver")
	if err != nil {
		return nil, fmt.Errorf("failed to stop game server: %s", strings.TrimSpace(string(output)))
	}
	return func() error {
		_, err := a.ConfigApp.Runner.CombinedOutput("/usr/bin/supervisorctl", "-c", "/etc/supervisor/conf.d/supervisord.conf", "start", "pzserver")
		return err
	}, nil
}

type normalizedBackupSettings struct {
	Enabled                    bool
	IntervalHours, MaxVersions int
}

func (a App) backupSettings() normalizedBackupSettings {
	settings := a.Settings.Load()
	interval, max := settings.BackupIntervalHours, settings.BackupMaxVersions
	if interval < 1 {
		interval = 24
	}
	if max < 1 {
		max = 10
	}
	if max > 100 {
		max = 100
	}
	return normalizedBackupSettings{Enabled: settings.BackupEnabled, IntervalHours: interval, MaxVersions: max}
}

func (s normalizedBackupSettings) toPanelSettings(current PanelSettings) PanelSettings {
	current.BackupEnabled, current.BackupIntervalHours, current.BackupMaxVersions = s.Enabled, s.IntervalHours, s.MaxVersions
	return current
}

func (a App) startBackupScheduler() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			settings := a.backupSettings()
			if !settings.Enabled {
				continue
			}
			persisted := a.Settings.Load()
			if persisted.BackupLastRunUnix == 0 {
				persisted.BackupLastRunUnix = time.Now().Unix()
				_ = a.Settings.Save(persisted)
				continue
			}
			lastRun := time.Unix(persisted.BackupLastRunUnix, 0)
			if time.Since(lastRun) < time.Duration(settings.IntervalHours)*time.Hour {
				continue
			}
			resume, err := a.stopGameForOperation()
			if err != nil {
				continue
			}
			_, err = a.Backups.Create("自动备份", readGameVersion(a.BaseDataDir, a.LogPath), settings.MaxVersions)
			if resume != nil {
				if startErr := resume(); err == nil {
					err = startErr
				}
			}
			if err == nil {
				persisted.BackupLastRunUnix = time.Now().Unix()
				_ = a.Settings.Save(persisted)
			}
		}
	}()
}
