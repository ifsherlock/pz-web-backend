package httpserver

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"pz-web-backend/internal/application/configapp"
	"pz-web-backend/internal/config"
)

// MemoryLimitKey 是面板里的"虚拟"配置项键名：不会写入 servertest.ini，
// 而是保存到 panel_settings.json，由 start-pz.sh 读取。
const MemoryLimitKey = "PZ_MEMORY_LIMIT"

func (a App) handleGetServerConfig(c *gin.Context) {
	lang := strings.ToUpper(c.DefaultQuery("lang", "CN"))
	lang = a.I18nApp.ResolveLang(lang)
	items, err := a.ConfigApp.GetServerConfig(lang)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 追加内存设置虚拟项：不写入 servertest.ini，仅面板展示，
	// 保存时由 handleSaveConfig 抽离并写入 panel_settings.json。
	// Section 直接用中文，确保前端分组标题显示"服务端配置"。
	if a.Settings != nil {
		settings := a.Settings.Load()
		items = append(items, config.Item{
			Key:     MemoryLimitKey,
			Value:   settings.MemoryLimit,
			Label:   "JVM 内存上限 (如 3g / 4g)",
			Tooltip: "游戏服务端 JVM 堆上限，重启游戏后生效；Build 42 建议 ≥ 2g",
			Section: "服务端配置",
		})
	}
	filename := fmt.Sprintf("%s.ini", a.ConfigApp.ServerName)
	c.JSON(http.StatusOK, gin.H{"filename": filename, "lang": lang, "items": items})
}

func (a App) handleGetSandboxConfig(c *gin.Context) {
	lang := strings.ToUpper(c.DefaultQuery("lang", "CN"))
	lang = a.I18nApp.ResolveLang(lang)
	items, err := a.ConfigApp.GetSandboxConfig(lang)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	filename := fmt.Sprintf("%s_SandboxVars.lua", a.ConfigApp.ServerName)
	c.JSON(http.StatusOK, gin.H{"filename": filename, "lang": lang, "items": items})
}

func (a App) handleSaveConfig(c *gin.Context) {
	name := configapp.SaveKind(c.Param("name"))
	if name != configapp.KindServer && name != configapp.KindSandbox {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config type"})
		return
	}

	var req SaveRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 抽离内存设置虚拟项：保存到 panel_settings.json，不写入 servertest.ini。
	toSave := req.Items
	if a.Settings != nil && name == configapp.KindServer {
		settings := a.Settings.Load()
		filtered := make([]config.Item, 0, len(req.Items))
		for _, item := range req.Items {
			if item.Key == MemoryLimitKey {
				memoryLimit, err := normalizeMemoryLimit(item.Value)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				settings.MemoryLimit = memoryLimit
				continue
			}
			filtered = append(filtered, item)
		}
		if err := a.Settings.Save(settings); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		toSave = filtered
	}

	if err := a.ConfigApp.Save(name, toSave, req.Restart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if req.Restart {
		c.JSON(http.StatusOK, gin.H{"status": "saved_and_restarting", "message": "Save completed! Restarting server..."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "saved", "message": "Successfully saved!"})
}
