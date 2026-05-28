package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ─── Importer ────────────────────────────────────────────

type Importer struct {
	sqlite    *gorm.DB      // 本地追踪库（必须）
	mysql     *gorm.DB      // 目标库（可为 nil）
	source    *SourceClient
	batchSize int

	mu          sync.Mutex
	activeBatch *ImportBatch   // 当前正在运行的批次
	cancelFunc  context.CancelFunc
}

func NewImporter(sqlite, mysql *gorm.DB, source *SourceClient, batchSize int) *Importer {
	return &Importer{
		sqlite:    sqlite,
		mysql:     mysql,
		source:    source,
		batchSize: batchSize,
	}
}

func (imp *Importer) MySQLAvailable() bool {
	if imp.mysql == nil {
		return false
	}
	db, err := imp.mysql.DB()
	if err != nil {
		return false
	}
	return db.Ping() == nil
}

// ─── 启动导入 ─────────────────────────────────────────────

type BatchConfig struct {
	BatchType string // test_1 / test_10 / test_100 / custom / full
	Limit     int    // 请求数量
	Offset    int    // 起始偏移
	Name      string
	Notes     string
}

func (imp *Importer) StartBatch(cfg BatchConfig) (int64, error) {
	imp.mu.Lock()
	defer imp.mu.Unlock()

	if imp.activeBatch != nil && imp.activeBatch.Status == "running" {
		return 0, fmt.Errorf("已有批次正在运行 (ID=%d)", imp.activeBatch.ID)
	}

	batch := &ImportBatch{
		Name:           cfg.Name,
		BatchType:      cfg.BatchType,
		SourceURL:      imp.source.BaseURL,
		OffsetStart:    cfg.Offset,
		RequestedCount: cfg.Limit,
		Status:         "running",
		Notes:          cfg.Notes,
	}
	if batch.Name == "" {
		batch.Name = fmt.Sprintf("%s @ %s", cfg.BatchType, time.Now().Format("01-02 15:04:05"))
	}
	if err := imp.sqlite.Create(batch).Error; err != nil {
		return 0, fmt.Errorf("创建批次记录失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	imp.cancelFunc = cancel
	imp.activeBatch = batch

	go imp.run(ctx, batch, cfg)
	return batch.ID, nil
}

func (imp *Importer) StopActive() {
	imp.mu.Lock()
	defer imp.mu.Unlock()
	if imp.cancelFunc != nil {
		imp.cancelFunc()
	}
}

func (imp *Importer) ActiveBatchID() int64 {
	imp.mu.Lock()
	defer imp.mu.Unlock()
	if imp.activeBatch != nil {
		return imp.activeBatch.ID
	}
	return 0
}

// ─── 核心导入流程 ─────────────────────────────────────────

func (imp *Importer) run(ctx context.Context, batch *ImportBatch, cfg BatchConfig) {
	defer func() {
		if r := recover(); r != nil {
			imp.failBatch(batch, fmt.Sprintf("panic: %v", r))
		}
	}()

	imp.batchLog(batch.ID, "info", fmt.Sprintf("批次 [%s] 开始，limit=%d offset=%d", cfg.BatchType, cfg.Limit, cfg.Offset))

	if !imp.MySQLAvailable() {
		imp.failBatch(batch, "MySQL 不可用，无法导入数据")
		return
	}

	totalPatients := 0
	totalExams := 0
	totalLis := 0
	totalErrors := 0

	remaining := cfg.Limit
	currentOffset := cfg.Offset

	for remaining > 0 {
		select {
		case <-ctx.Done():
			imp.batchLog(batch.ID, "warn", fmt.Sprintf("批次被停止 (offset=%d)", currentOffset))
			imp.finishBatch(batch, "stopped", totalPatients, totalExams, totalLis, totalErrors, "手动停止")
			return
		default:
		}

		fetchN := remaining
		if fetchN > imp.batchSize {
			fetchN = imp.batchSize
		}

		resp, err := imp.source.Batch(fetchN, currentOffset)
		if err != nil {
			imp.batchLog(batch.ID, "error", fmt.Sprintf("拉取数据失败 offset=%d: %v", currentOffset, err))
			imp.finishBatch(batch, "error", totalPatients, totalExams, totalLis, totalErrors, err.Error())
			return
		}
		if resp.Metadata.ExaminfoCount == 0 {
			imp.batchLog(batch.ID, "info", "数据已全部读取完毕")
			break
		}

		imp.batchLog(batch.ID, "info", fmt.Sprintf("处理 offset=%d examinfo=%d lis=%d",
			currentOffset, resp.Metadata.ExaminfoCount, resp.Metadata.LisCount))

		// 建 exam_id → LIS 映射
		lisMap := map[string][]SourceLisRecord{}
		for _, l := range resp.Lis {
			lisMap[l.ExamID] = append(lisMap[l.ExamID], l)
		}

		for _, raw := range resp.Examinfo {
			var rec SourceExaminfoRecord
			if err := json.Unmarshal(raw, &rec); err != nil {
				totalErrors++
				continue
			}
			rec.Normalize()
			if rec.ExamID == "" {
				totalErrors++
				continue
			}

			pid, examOK, lisN, err := imp.importOne(batch.ID, rec, lisMap[rec.ExamID])
			if err != nil {
				imp.batchLog(batch.ID, "warn", fmt.Sprintf("患者 %s 入库失败: %v", rec.ExamID, err))
				totalErrors++
				imp.sqlite.Create(&ImportBatchRecord{
					BatchID:   batch.ID,
					ExamID:    rec.ExamID,
					Status:    "error",
					ErrorMsg:  err.Error(),
				})
				continue
			}

			// 记录到 SQLite
			age := calcAge(normalizeDate(rec.BirthDate))
			imp.sqlite.Create(&ImportBatchRecord{
				BatchID:        batch.ID,
				ExamID:         rec.ExamID,
				PatientName:    rec.Name,
				PatientGender:  normalizeGender(rec.Gender),
				PatientAge:     age,
				ExamDate:       normalizeDate(rec.ExamDate),
				LisCount:       lisN,
				MySQLPatientID: int64(pid),
				Status:         "ok",
			})

			if examOK {
				totalExams++
			}
			totalPatients++
			totalLis += lisN
		}

		currentOffset += resp.Metadata.ExaminfoCount
		remaining -= resp.Metadata.ExaminfoCount

		if resp.Metadata.ExaminfoCount < fetchN {
			break
		}
	}

	imp.batchLog(batch.ID, "info", fmt.Sprintf("✅ 完成：患者=%d 体检=%d LIS=%d 错误=%d",
		totalPatients, totalExams, totalLis, totalErrors))
	imp.finishBatch(batch, "completed", totalPatients, totalExams, totalLis, totalErrors, "")

	imp.mu.Lock()
	imp.activeBatch = nil
	imp.mu.Unlock()
}

// importOne 导入单个患者，返回 (mysqlPatientID, examCreated, lisCount, error)
func (imp *Importer) importOne(batchID int64, rec SourceExaminfoRecord, lisRows []SourceLisRecord) (uint, bool, int, error) {
	examDate := normalizeDate(rec.ExamDate)
	if examDate == "" {
		examDate = time.Now().Format("2006-01-02")
	}
	age := calcAge(normalizeDate(rec.BirthDate))
	gender := normalizeGender(rec.Gender)
	bd := normalizeDate(rec.BirthDate)

	// ── patients ──
	p := DBPatient{
		PatientID:  rec.ExamID,
		Name:       rec.Name,
		Gender:     gender,
		DataSource: "体检",
	}
	if bd != "" {
		p.BirthDate = &bd
	}
	if age > 0 {
		p.Age = &age
	}

	res := imp.mysql.Where("patient_id = ?", rec.ExamID).FirstOrCreate(&p)
	if res.Error != nil {
		return 0, false, 0, fmt.Errorf("patients upsert: %w", res.Error)
	}

	// ── patient_modules(stroke) ──
	pm := DBPatientModule{PatientID: int(p.ID), ModuleCode: "stroke", Status: "active"}
	imp.mysql.Where("patient_id = ? AND module_code = ?", p.ID, "stroke").
		Attrs(pm).FirstOrCreate(&pm)

	// ── health_examinations ──
	exam := DBHealthExamination{
		ExamID:    rec.ExamID,
		ExamDate:  examDate,
		PatientID: int(p.ID),
	}
	examRes := imp.mysql.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "exam_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"exam_date", "patient_id", "updated_at"}),
	}).Create(&exam)
	if examRes.Error != nil {
		return p.ID, false, 0, fmt.Errorf("health_examinations upsert: %w", examRes.Error)
	}
	examCreated := examRes.RowsAffected > 0

	// ── exam_results：体征 ──
	vitals := buildVitals(rec)
	for _, v := range vitals {
		v.ExamID = rec.ExamID
		v.PatientID = int(p.ID)
		imp.mysql.Create(&v)
	}

	// ── exam_results：LIS ──
	lisCount := 0
	for _, lis := range lisRows {
		if lis.Jczbmc == "" {
			continue
		}
		isAbnormal := strings.TrimSpace(lis.Ycts) != ""
		r := DBExamResult{
			ExamID:         rec.ExamID,
			ItemCode:       strPtr(lis.Jcxmbm),
			ItemName:       lis.Jczbmc,
			ResultValue:    strPtr(lis.Jczbjg),
			Unit:           strPtr(lis.Jldw),
			ReferenceRange: strPtr(lis.Ckz),
			IsAbnormal:     isAbnormal,
			Category:       strPtr(lis.Jcxmdlmc),
			PatientID:      int(p.ID),
		}
		if err := imp.mysql.Create(&r).Error; err == nil {
			lisCount++
		}
	}

	return p.ID, examCreated, lisCount, nil
}

// ─── 撤销批次 ─────────────────────────────────────────────

type RollbackResult struct {
	BatchID      int64    `json:"batch_id"`
	PatientsIDs  []int64  `json:"patient_ids"`
	DeletedRows  map[string]int64 `json:"deleted_rows"`
	Errors       []string `json:"errors"`
}

func (imp *Importer) Rollback(batchID int64) (*RollbackResult, error) {
	// 检查批次
	var batch ImportBatch
	if err := imp.sqlite.First(&batch, batchID).Error; err != nil {
		return nil, fmt.Errorf("批次不存在: %w", err)
	}
	if batch.Status == "rolled_back" {
		return nil, fmt.Errorf("批次已撤销")
	}
	if batch.Status == "running" {
		return nil, fmt.Errorf("批次仍在运行，请先停止")
	}
	if !imp.MySQLAvailable() {
		return nil, fmt.Errorf("MySQL 不可用，无法执行撤销")
	}

	// 获取该批次的所有 MySQL patient_id
	var records []ImportBatchRecord
	imp.sqlite.Where("batch_id = ? AND status = ? AND mysql_patient_id > 0", batchID, "ok").Find(&records)

	if len(records) == 0 {
		// 仍标记为 rolled_back
		now := time.Now()
		imp.sqlite.Model(&batch).Updates(map[string]any{"status": "rolled_back", "rolled_back_at": now})
		return &RollbackResult{BatchID: batchID, DeletedRows: map[string]int64{}}, nil
	}

	patientIDs := make([]int64, 0, len(records))
	for _, r := range records {
		if r.MySQLPatientID > 0 {
			patientIDs = append(patientIDs, r.MySQLPatientID)
		}
	}

	result := &RollbackResult{
		BatchID:     batchID,
		PatientsIDs: patientIDs,
		DeletedRows: map[string]int64{},
		Errors:      []string{},
	}

	// 按表顺序删除
	for _, table := range rollbackTables {
		col := "patient_id"
		if table == "patients" {
			col = "id"
		}
		q := imp.mysql.Exec(
			fmt.Sprintf("DELETE FROM `%s` WHERE `%s` IN ?", table, col),
			patientIDs,
		)
		if q.Error != nil {
			// 表可能不存在，记录 warning 但继续
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", table, q.Error))
		} else {
			result.DeletedRows[table] = q.RowsAffected
		}
	}

	now := time.Now()
	imp.sqlite.Model(&batch).Updates(map[string]any{
		"status":         "rolled_back",
		"rolled_back_at": now,
	})

	return result, nil
}

// ─── 辅助 ─────────────────────────────────────────────────

func (imp *Importer) batchLog(batchID int64, level, msg string) {
	imp.sqlite.Create(&ImportBatchLog{BatchID: batchID, Level: level, Message: msg})
}

func (imp *Importer) failBatch(batch *ImportBatch, msg string) {
	imp.sqlite.Model(batch).Updates(map[string]any{
		"status":        "error",
		"error_summary": msg,
		"finished_at":   time.Now(),
	})
	imp.batchLog(batch.ID, "error", msg)
	imp.mu.Lock()
	imp.activeBatch = nil
	imp.mu.Unlock()
}

func (imp *Importer) finishBatch(batch *ImportBatch, status string,
	patients, exams, lis, errors int, errMsg string) {

	now := time.Now()
	imp.sqlite.Model(batch).Updates(map[string]any{
		"status":          status,
		"actual_patients": patients,
		"actual_exams":    exams,
		"actual_lis":      lis,
		"actual_errors":   errors,
		"finished_at":     now,
		"error_summary":   errMsg,
	})
}

// ─── 工具函数 ─────────────────────────────────────────────

func normalizeGender(g string) string {
	switch strings.TrimSpace(g) {
	case "1", "男", "Male", "M", "male":
		return "Male"
	case "2", "女", "Female", "F", "female":
		return "Female"
	default:
		return g
	}
}

func normalizeDate(d string) string {
	d = strings.TrimSpace(d)
	if d == "" {
		return ""
	}
	d = strings.ReplaceAll(d, "/", "-")
	if len(d) == 8 && !strings.Contains(d, "-") {
		return d[:4] + "-" + d[4:6] + "-" + d[6:]
	}
	if len(d) > 10 {
		return d[:10]
	}
	return d
}

func calcAge(birthDate string) int {
	if birthDate == "" {
		return 0
	}
	t, err := time.Parse("2006-01-02", birthDate)
	if err != nil {
		return 0
	}
	age := time.Now().Year() - t.Year()
	if time.Now().YearDay() < t.YearDay() {
		age--
	}
	if age < 0 || age > 150 {
		return 0
	}
	return age
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func floatStr(f float64) *string {
	if f == 0 {
		return nil
	}
	s := strconv.FormatFloat(f, 'f', 2, 64)
	return &s
}

func coalesce(cn string, num float64) *string {
	if cn != "" {
		return &cn
	}
	return floatStr(num)
}

func buildVitals(rec SourceExaminfoRecord) []DBExamResult {
	type item struct{ name string; val *string; unit string }
	items := []item{
		{"身高", coalesce(rec.HeightCN, 0), "cm"},
		{"体重", coalesce(rec.WeightCN, 0), "kg"},
		{"腰围", coalesce(rec.WaistCN, rec.Waistline), "cm"},
		{"BMI", coalesce(rec.BMICN, rec.BMI), "kg/m²"},
		{"收缩压(左)", coalesce(rec.LeftSBPCN, rec.LeftSBP), "mmHg"},
		{"舒张压(左)", coalesce(rec.LeftDBPCN, rec.LeftDBP), "mmHg"},
		{"收缩压(右)", coalesce(rec.RightSBPCN, rec.RightSBP), "mmHg"},
		{"舒张压(右)", coalesce(rec.RightDBPCN, rec.RightDBP), "mmHg"},
		{"收缩压", coalesce(rec.SBPCN, 0), "mmHg"},
		{"舒张压", coalesce(rec.DBPCN, 0), "mmHg"},
	}
	cat := strPtr("体征")
	var out []DBExamResult
	for _, v := range items {
		if v.val == nil {
			continue
		}
		u := v.unit
		out = append(out, DBExamResult{
			ItemName: v.name, ResultValue: v.val,
			Unit: &u, Category: cat,
		})
	}
	return out
}
