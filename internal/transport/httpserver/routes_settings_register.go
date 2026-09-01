package httpserver

import "github.com/gin-gonic/gin"

func (a App) registerSettingsRoutes(r *gin.Engine) {
	r.GET("/api/settings", a.handleGetSettings)
	r.POST("/api/settings", a.handleSaveSettings)
}
