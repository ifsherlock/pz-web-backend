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
const GameBranchKey = "PZ_BRANCH"
const AdminUsernameKey = "PZ_ADMIN_USERNAME"
const AdminPasswordKey = "PZ_ADMIN_PASSWORD"

func (a App) handleGetServerConfig(c *gin.Context) {
	lang := strings.ToUpper(c.DefaultQuery("lang", "CN"))
	lang = a.I18nApp.ResolveLang(lang)
	items, err := a.ConfigApp.GetServerConfig(lang)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 将密码相关配置集中到末尾的安全分组，便于在同一处维护入服与管理员凭据。
	securityKeys := map[string]bool{"Password": true, "ServerPassword": true, "RCONPassword": true}
	normalItems := make([]config.Item, 0, len(items))
	securityItems := make([]config.Item, 0, 3)
	for _, item := range items {
		if securityKeys[item.Key] {
			item.Section = "服务端安全"
			securityItems = append(securityItems, item)
			continue
		}
		normalItems = append(normalItems, item)
	}
	items = append(normalItems, securityItems...)
	// 追加内存设置虚拟项：不写入 servertest.ini，仅面板展示，
	// 保存时由 handleSaveConfig 抽离并写入 panel_settings.json。
	// Section 直接用中文，确保前端分组标题显示"服务端配置"。
	if a.Settings != nil {
		settings := a.Settings.Load()
		branch, _ := normalizeGameBranch(settings.GameBranch)
		items = append(items, config.Item{
			Key:     MemoryLimitKey,
			Value:   settings.MemoryLimit,
			Label:   "JVM 内存上限 (如 3g / 4g)",
			Tooltip: "游戏服务端 JVM 堆上限，重启游戏后生效；Build 42 建议 ≥ 2g",
			Section: "服务端配置",
		})
		items = append(items, config.Item{
			Key:     GameBranchKey,
			Value:   branch,
			Label:   "游戏版本分支",
			Tooltip: "public 当前最新为 42.20.4，服务端启动时会自动拉取该分支最新内容；也可输入 Steam 官方分支名。SteamCMD 不能直接用不存在的版本号下载。",
			Section: "服务端配置",
			Options: []config.Option{{Value: "public", Label: "稳定版 42.20.4 (public，自动更新)"}, {Value: "42.19", Label: "42.19.2 (42.19，自动更新)"}, {Value: "legacy41", Label: "41.78.21 (legacy41，自动更新)"}, {Value: "__custom__", Label: "自定义 Steam 分支名"}},
		})
		adminUsername, _ := normalizeAdminUsername(settings.AdminUsername)
		adminPassword := ""
		if settings.AdminPassword != "" {
			adminPassword = "********"
		}
		items = append(items, config.Item{Key: AdminUsernameKey, Value: adminUsername, Label: "管理员账户", Tooltip: "游戏内管理员账户名；保存并重启后生效。", Section: "服务端安全"})
		items = append(items, config.Item{Key: AdminPasswordKey, Value: adminPassword, Label: "管理员密码", Tooltip: "游戏内管理员密码；留空表示保持当前密码，保存并重启后生效。", Section: "服务端安全"})
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
			if item.Key == GameBranchKey {
				branch, err := normalizeGameBranch(item.Value)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				settings.GameBranch = branch
				continue
			}
			if item.Key == AdminUsernameKey {
				username, err := normalizeAdminUsername(item.Value)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				settings.AdminUsername = username
				continue
			}
			if item.Key == AdminPasswordKey {
				if item.Value != "" && item.Value != "********" {
					if strings.ContainsAny(item.Value, "\r\n") || len(item.Value) < 4 || len(item.Value) > 128 {
						c.JSON(http.StatusBadRequest, gin.H{"error": "admin password must be 4-128 characters without line breaks"})
						return
					}
					settings.AdminPassword = item.Value
				}
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
