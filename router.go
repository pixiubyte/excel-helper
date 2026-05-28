package main

import (
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (a *App) registerRoutes(r *gin.Engine) {
	tmpl := template.Must(template.ParseFS(embeddedFiles, "templates/index.html"))
	r.SetHTMLTemplate(tmpl)
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{"title": "Examinfo/LIS 数据浏览"})
	})
	r.GET("/favicon.ico", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.GET("/api/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	r.POST("/api/scan", a.handleScan)
	r.GET("/api/scan/status", a.handleScanStatus)
	r.GET("/api/stats", a.handleStats)
	r.GET("/api/patients", a.handlePatients)
	r.GET("/api/patients/:examID", a.handlePatientDetail)
	r.GET("/api/stroke/batch", a.handleStrokeBatch)
	r.GET("/api/stroke/patient/:examID", a.handleStrokePatient)
}
