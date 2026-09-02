package httpserver

import (
	"io/fs"

	"github.com/gin-gonic/gin"
)

type Config struct {
	BaseDataDir string
	BaseGameDir string
	ServerName  string
	LogPath     string
	DevMode     bool
	Build       BuildInfo

	ContentFS fs.FS
}

func NewEngine(cfg Config) *gin.Engine {
	r := gin.Default()

	// 禁用浏览器缓存：确保部署后用户立即看到最新界面，避免强刷才生效。
	r.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Next()
	})

	SetupStaticAndTemplates(r, cfg.ContentFS)

	app := NewApp(cfg.BaseDataDir, cfg.BaseGameDir, cfg.ServerName, cfg.LogPath, cfg.Build, cfg.DevMode)
	app.RegisterRoutes(r)
	if !cfg.DevMode {
		app.startBackupScheduler()
	}
	return r
}
