package httpserver

import "github.com/gin-gonic/gin"

func (a App) registerBackupRoutes(r *gin.Engine) {
	r.GET("/api/backups", a.handleGetBackupSettings)
	r.POST("/api/backups/settings", a.handleSaveBackupSettings)
	r.POST("/api/backups", a.handleCreateBackup)
	r.DELETE("/api/backups/:id", a.handleDeleteBackup)
	r.POST("/api/backups/:id/restore", a.handleRestoreBackup)
}
