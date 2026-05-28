package main

import (
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (a *App) registerRoutes(r *gin.Engine) {
	tmpl := template.Must(template.ParseFS(embeddedFiles, "templates/index.html"))
	r.SetHTMLTemplate(tmpl)

	r.GET("/", a.handleIndex)
	r.GET("/favicon.ico", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	api := r.Group("/api")
	{
		api.GET("/status", a.handleStatus)

		// 数据源
		api.GET("/source/stats", a.handleSourceStats)
		api.GET("/source/scan-status", a.handleSourceScanStatus)
		api.GET("/source/patients", a.handleSourcePatients)

		// 导入控制
		api.POST("/import/start", a.handleImportStart)
		api.POST("/import/stop", a.handleImportStop)

		// 批次管理（从 SQLite 读，无需 MySQL）
		api.GET("/batches", a.handleBatches)
		api.GET("/batches/:id", a.handleBatchDetail)
		api.POST("/batches/:id/rollback", a.handleRollback)

		// MySQL 统计
		api.GET("/db/stats", a.handleDBStats)
	}
}
