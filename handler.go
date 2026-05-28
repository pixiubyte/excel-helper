package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GET /  → index.html
func (a *App) handleIndex(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{"title": "数据导入工具"})
}

// GET /api/status  —— 服务整体状态
func (a *App) handleStatus(c *gin.Context) {
	mysqlOK := a.importer.MySQLAvailable()
	sourceOK := a.importer.source.Health()

	c.JSON(http.StatusOK, gin.H{
		"mysql_ok":     mysqlOK,
		"source_ok":    sourceOK,
		"source_url":   a.importer.source.BaseURL,
		"active_batch": a.importer.ActiveBatchID(),
		"time":         time.Now().Format(time.RFC3339),
	})
}

// GET /api/source/stats
func (a *App) handleSourceStats(c *gin.Context) {
	stats, err := a.importer.source.Stats()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GET /api/source/scan-status
func (a *App) handleSourceScanStatus(c *gin.Context) {
	s, err := a.importer.source.ScanStatus()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s)
}

// GET /api/source/patients?page=1&page_size=20&q=
func (a *App) handleSourcePatients(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	q := c.Query("q")
	list, err := a.importer.source.Patients(page, pageSize, q)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// POST /api/import/start
// body: { "batch_type":"test_1|test_10|test_100|full|custom", "limit":N, "offset":0, "name":"", "notes":"" }
func (a *App) handleImportStart(c *gin.Context) {
	var req struct {
		BatchType string `json:"batch_type"`
		Limit     int    `json:"limit"`
		Offset    int    `json:"offset"`
		Name      string `json:"name"`
		Notes     string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 推导 limit
	switch req.BatchType {
	case "test_1":
		req.Limit = 1
	case "test_10":
		req.Limit = 10
	case "test_100":
		req.Limit = 100
	case "full":
		req.Limit = 999999
	}
	if req.Limit <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit 必须 > 0"})
		return
	}

	batchID, err := a.importer.StartBatch(BatchConfig{
		BatchType: req.BatchType,
		Limit:     req.Limit,
		Offset:    req.Offset,
		Name:      req.Name,
		Notes:     req.Notes,
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"batch_id": batchID, "message": "导入已启动"})
}

// POST /api/import/stop
func (a *App) handleImportStop(c *gin.Context) {
	a.importer.StopActive()
	c.JSON(http.StatusOK, gin.H{"message": "停止信号已发送"})
}

// GET /api/batches?page=1
func (a *App) handleBatches(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	var batches []ImportBatch
	a.sqlite.Order("id desc").Limit(30).Offset((page - 1) * 30).Find(&batches)
	var total int64
	a.sqlite.Model(&ImportBatch{}).Count(&total)
	c.JSON(http.StatusOK, gin.H{"total": total, "page": page, "items": batches})
}

// GET /api/batches/:id
func (a *App) handleBatchDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	var batch ImportBatch
	if err := a.sqlite.First(&batch, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "批次不存在"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := 50
	var records []ImportBatchRecord
	a.sqlite.Where("batch_id = ?", id).
		Order("id asc").Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&records)
	var recTotal int64
	a.sqlite.Model(&ImportBatchRecord{}).Where("batch_id = ?", id).Count(&recTotal)

	var logs []ImportBatchLog
	a.sqlite.Where("batch_id = ?", id).Order("id asc").Limit(200).Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"batch":   batch,
		"records": gin.H{"total": recTotal, "page": page, "items": records},
		"logs":    logs,
	})
}

// POST /api/batches/:id/rollback
func (a *App) handleRollback(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	result, err := a.importer.Rollback(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /api/db/stats  —— MySQL 已入库统计
func (a *App) handleDBStats(c *gin.Context) {
	if !a.importer.MySQLAvailable() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "MySQL 不可用"})
		return
	}
	var pCount, examCount, resultCount int64
	a.importer.mysql.Model(&DBPatient{}).Count(&pCount)
	a.importer.mysql.Model(&DBHealthExamination{}).Count(&examCount)
	a.importer.mysql.Model(&DBExamResult{}).Count(&resultCount)

	// 评估数量
	assessCounts := map[string]int64{}
	for _, t := range []string{"fsp_risk_assessments_json", "ascvd_risk_assessments_json", "china_par_assessments_json"} {
		var cnt int64
		a.importer.mysql.Raw("SELECT COUNT(*) FROM `"+t+"`").Scan(&cnt)
		assessCounts[t] = cnt
	}

	c.JSON(http.StatusOK, gin.H{
		"patients":            pCount,
		"health_examinations": examCount,
		"exam_results":        resultCount,
		"assessments":         assessCounts,
	})
}
