package httpserver

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// handleGetSettings 返回面板设置(Steam API Key / 内存上限)。
// 不回传完整 Key，仅返回是否已配置与掩码，便于前端确认存放成功。
func (a App) handleGetSettings(c *gin.Context) {
	settings := a.Settings.Load()
	c.JSON(http.StatusOK, gin.H{
		"steam_api_key_configured": settings.SteamAPIKey != "",
		"steam_api_key_masked":     maskKey(settings.SteamAPIKey),
		"memory_limit":             settings.MemoryLimit,
	})
}

// maskKey 将 Key 掩码为前4位+后4位，中间用 * 隐藏。
func maskKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// handleSaveSettings 保存面板设置。
// 采用指针字段：仅更新请求中显式传入的字段，避免覆盖未提交的配置。
func (a App) handleSaveSettings(c *gin.Context) {
	var req struct {
		SteamAPIKey *string `json:"steam_api_key"`
		MemoryLimit *string `json:"memory_limit"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	next := a.Settings.Load()
	if req.SteamAPIKey != nil {
		next.SteamAPIKey = *req.SteamAPIKey
	}
	if req.MemoryLimit != nil {
		next.MemoryLimit = *req.MemoryLimit
	}

	if err := a.Settings.Save(next); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 同步更新 WorkshopClient 的 API Key，合集展开即时生效(无需重启面板)。
	if req.SteamAPIKey != nil {
		if ws, ok := a.ModsApp.Workshop.(interface{ SetSteamAPIKey(string) }); ok {
			ws.SetSteamAPIKey(next.SteamAPIKey)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":                 "ok",
		"steam_api_key_masked":   maskKey(next.SteamAPIKey),
		"memory_limit":           next.MemoryLimit,
	})
}
