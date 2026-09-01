package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleGetSettings 返回面板设置(Steam API Key / 内存上限)。
func (a App) handleGetSettings(c *gin.Context) {
	settings := a.Settings.Load()
	// 不回传完整 Key，仅返回是否已配置
	c.JSON(http.StatusOK, gin.H{
		"steam_api_key_configured": settings.SteamAPIKey != "",
		"memory_limit":             settings.MemoryLimit,
	})
}

// handleSaveSettings 保存面板设置。
func (a App) handleSaveSettings(c *gin.Context) {
	var req struct {
		SteamAPIKey string `json:"steam_api_key"`
		MemoryLimit string `json:"memory_limit"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	current := a.Settings.Load()
	// 允许传空字符串来清除，不覆盖未提供的字段
	next := current
	next.SteamAPIKey = req.SteamAPIKey
	next.MemoryLimit = req.MemoryLimit

	if err := a.Settings.Save(next); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 更新 WorkshopClient 的 API Key，合集展开即时生效(无需重启面板)。
	if ws, ok := a.ModsApp.Workshop.(interface{ SetSteamAPIKey(string) }); ok {
		ws.SetSteamAPIKey(req.SteamAPIKey)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
