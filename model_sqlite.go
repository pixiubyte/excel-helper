package main

import "time"

// ─── SQLite 表（本地 import 跟踪，无需 MySQL）──────────────

// ImportBatch 一次导入操作的批次记录
type ImportBatch struct {
	ID             int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name           string     `gorm:"column:name" json:"name"`
	BatchType      string     `gorm:"column:batch_type" json:"batch_type"` // test_1/test_10/test_100/full/custom
	SourceURL      string     `gorm:"column:source_url" json:"source_url"`
	OffsetStart    int        `gorm:"column:offset_start" json:"offset_start"`
	RequestedCount int        `gorm:"column:requested_count" json:"requested_count"`
	ActualPatients int        `gorm:"column:actual_patients;default:0" json:"actual_patients"`
	ActualExams    int        `gorm:"column:actual_exams;default:0" json:"actual_exams"`
	ActualLis      int        `gorm:"column:actual_lis;default:0" json:"actual_lis"`
	ActualErrors   int        `gorm:"column:actual_errors;default:0" json:"actual_errors"`
	// Status: running / completed / error / rolled_back / stopped
	Status       string     `gorm:"column:status;default:running" json:"status"`
	ErrorSummary string     `gorm:"column:error_summary" json:"error_summary"`
	StartedAt    time.Time  `gorm:"column:started_at;autoCreateTime" json:"started_at"`
	FinishedAt   *time.Time `gorm:"column:finished_at" json:"finished_at,omitempty"`
	RolledBackAt *time.Time `gorm:"column:rolled_back_at" json:"rolled_back_at,omitempty"`
	Notes        string     `gorm:"column:notes" json:"notes"`
}

func (ImportBatch) TableName() string { return "import_batches" }

// ImportBatchRecord 批次内每个患者的导入记录（存 MySQL ID 用于撤销）
type ImportBatchRecord struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	BatchID        int64     `gorm:"column:batch_id;index" json:"batch_id"`
	ExamID         string    `gorm:"column:exam_id" json:"exam_id"`           // 源 exam_id
	PatientName    string    `gorm:"column:patient_name" json:"patient_name"`
	PatientGender  string    `gorm:"column:patient_gender" json:"patient_gender"`
	PatientAge     int       `gorm:"column:patient_age" json:"patient_age"`
	ExamDate       string    `gorm:"column:exam_date" json:"exam_date"`
	LisCount       int       `gorm:"column:lis_count;default:0" json:"lis_count"`
	MySQLPatientID int64     `gorm:"column:mysql_patient_id;default:0" json:"mysql_patient_id"` // patients.id
	Status         string    `gorm:"column:status;default:ok" json:"status"`  // ok / error
	ErrorMsg       string    `gorm:"column:error_msg" json:"error_msg"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (ImportBatchRecord) TableName() string { return "import_batch_records" }

// ImportBatchLog 每批次的详细日志行
type ImportBatchLog struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	BatchID   int64     `gorm:"column:batch_id;index" json:"batch_id"`
	Level     string    `gorm:"column:level;default:info" json:"level"` // info/warn/error
	Message   string    `gorm:"column:message" json:"message"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (ImportBatchLog) TableName() string { return "import_batch_logs" }
