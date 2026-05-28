package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (a *App) handleScan(c *gin.Context) {
	a.mu.Lock()
	if a.status.Running {
		a.mu.Unlock()
		c.JSON(http.StatusConflict, gin.H{"error": "扫描正在进行"})
		return
	}
	a.status = &ScanStatus{Running: true, StartedAt: time.Now(), Message: "准备扫描"}
	a.mu.Unlock()

	go func() {
		if err := a.scanAll(); err != nil {
			a.mu.Lock()
			a.status.Running = false
			a.status.FinishedAt = time.Now()
			a.status.ErrorMessage = err.Error()
			a.status.Message = "扫描失败"
			a.mu.Unlock()
			log.Printf("scan failed: %v", err)
			return
		}
		a.mu.Lock()
		a.status.Running = false
		a.status.FinishedAt = time.Now()
		a.status.Message = fmt.Sprintf(
			"扫描完成 | examinfo 共计%d张表，共%d条数据，已入库%d条 | lis 共计%d张表，共%d条数据，已入库%d条",
			a.status.ExamFiles, a.status.ExamRawRows, a.status.ExamRows,
			a.status.LisFiles, a.status.LisRawRows, a.status.LisRows,
		)
		a.mu.Unlock()
	}()
	c.JSON(http.StatusAccepted, a.status)
}

func (a *App) handleScanStatus(c *gin.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	c.JSON(http.StatusOK, a.status)
}

func (a *App) handleStats(c *gin.Context) {
	var patientCount, lisLinked, lisRows, abnormalRows int64
	_ = a.db.Model(&Patient{}).Count(&patientCount).Error
	_ = a.db.Model(&LISSummary{}).Where("lis_count > ?", 0).Count(&lisLinked).Error
	var summaries []LISSummary
	_ = a.db.Find(&summaries).Error
	for _, item := range summaries {
		lisRows += int64(item.LisCount)
		abnormalRows += int64(item.AbnormalCount)
	}
	c.JSON(http.StatusOK, gin.H{
		"patients":          patientCount,
		"patients_with_lis": lisLinked,
		"lis_rows":          lisRows,
		"abnormal_rows":     abnormalRows,
		"data_dir":          a.dataDir,
	})
}

func (a *App) handlePatients(c *gin.Context) {
	page := positiveInt(c.Query("page"), 1)
	pageSize := clamp(positiveInt(c.Query("page_size"), 50), 1, 500)
	q := strings.TrimSpace(c.Query("q"))
	offset := (page - 1) * pageSize

	query := a.db.Model(&Patient{})
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("exam_id LIKE ? OR name LIKE ?", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var patients []Patient
	if err := query.Order("exam_id").Limit(pageSize).Offset(offset).Find(&patients).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	examIDs := make([]string, 0, len(patients))
	for _, patient := range patients {
		examIDs = append(examIDs, patient.ExamID)
	}
	summaryMap := map[string]LISSummary{}
	if len(examIDs) > 0 {
		var summaries []LISSummary
		if err := a.db.Where("exam_id IN ?", examIDs).Find(&summaries).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for _, summary := range summaries {
			summaryMap[summary.ExamID] = summary
		}
	}
	items := []PatientListItem{}
	for _, patient := range patients {
		summary := summaryMap[patient.ExamID]
		items = append(items, PatientListItem{
			ExamID:         patient.ExamID,
			YLJGDM:         patient.YLJGDM,
			Name:           patient.Name,
			Gender:         patient.Gender,
			BirthDate:      patient.BirthDate,
			ExamDate:       patient.ExamDate,
			LeftSBP:        patient.LeftSBP,
			LeftDBP:        patient.LeftDBP,
			RightSBP:       patient.RightSBP,
			RightDBP:       patient.RightDBP,
			Waistline:      patient.Waistline,
			BMI:            patient.BMI,
			LisCount:       summary.LisCount,
			AbnormalCount:  summary.AbnormalCount,
			ExamSourceFile: patient.ExamSourceFile,
			FirstLisFile:   summary.FirstFile,
			LastLisFile:    summary.LastFile,
		})
	}
	c.JSON(http.StatusOK, gin.H{"page": page, "page_size": pageSize, "total": total, "items": items})
}

func (a *App) handlePatientDetail(c *gin.Context) {
	examID := c.Param("examID")
	patient, err := a.getPatientRaw(examID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	lis, err := a.findLisRows(examID, 5000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"examinfo": patient, "lis": lis, "lis_count": len(lis)})
}

func (a *App) handleStrokePatient(c *gin.Context) {
	a.handlePatientDetail(c)
}

func (a *App) handleStrokeBatch(c *gin.Context) {
	limit := clamp(positiveInt(c.Query("limit"), 50), 1, 200)
	offset := positiveInt(c.Query("offset"), 0)

	var patients []Patient
	if err := a.db.Order("exam_id").Limit(limit).Offset(offset).Find(&patients).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	examinfo := []map[string]any{}
	examIDs := []string{}
	for _, patient := range patients {
		var m map[string]any
		if err := json.Unmarshal([]byte(patient.RawJSON), &m); err == nil {
			examIDs = append(examIDs, patient.ExamID)
			examinfo = append(examinfo, m)
		}
	}
	lis := []map[string]any{}
	for _, id := range examIDs {
		rows, err := a.findLisRows(id, 100)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		lis = append(lis, rows...)
	}
	c.JSON(http.StatusOK, gin.H{"metadata": gin.H{"limit": limit, "offset": offset, "examinfo_count": len(examinfo), "lis_count": len(lis)}, "examinfo": examinfo, "lis": lis})
}
