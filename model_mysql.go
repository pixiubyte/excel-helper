package main

import "time"

// ─── MySQL 表（stroke-demo 数据库）───────────────────────

type DBPatient struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	PatientID  string    `gorm:"column:patient_id;uniqueIndex;not null;size:50" json:"patient_id"`
	Name       string    `gorm:"column:name;not null;size:100" json:"name"`
	Gender     string    `gorm:"column:gender;not null;size:10" json:"gender"`
	BirthDate  *string   `gorm:"column:birth_date" json:"birth_date,omitempty"`
	Age        *int      `gorm:"column:age" json:"age,omitempty"`
	DataSource string    `gorm:"column:data_source;default:体检;size:20" json:"data_source"`
	IDCard     *string   `gorm:"column:id_card;uniqueIndex;size:18" json:"id_card,omitempty"`
	Version    int       `gorm:"column:version;default:0" json:"-"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (DBPatient) TableName() string { return "patients" }

type DBPatientModule struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	PatientID  int       `gorm:"column:patient_id;not null;uniqueIndex:upm"`
	ModuleCode string    `gorm:"column:module_code;not null;size:20;uniqueIndex:upm"`
	Status     string    `gorm:"column:status;default:active"`
	EnrolledAt time.Time `gorm:"column:enrolled_at;autoCreateTime"`
}

func (DBPatientModule) TableName() string { return "patient_modules" }

type DBHealthExamination struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ExamID    string    `gorm:"column:exam_id;uniqueIndex;not null;size:255" json:"exam_id"`
	ExamDate  string    `gorm:"column:exam_date;not null" json:"exam_date"`
	ExamTime  *string   `gorm:"column:exam_time" json:"exam_time,omitempty"`
	ExamType  *string   `gorm:"column:exam_type;size:255" json:"exam_type,omitempty"`
	PatientID int       `gorm:"column:patient_id;not null;index" json:"patient_id"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (DBHealthExamination) TableName() string { return "health_examinations" }

type DBExamResult struct {
	ID             uint      `gorm:"primaryKey;autoIncrement"`
	ExamID         string    `gorm:"column:exam_id;not null;size:255;index"`
	ItemCode       *string   `gorm:"column:item_code;size:20"`
	ItemName       string    `gorm:"column:item_name;not null;size:100"`
	ResultValue    *string   `gorm:"column:result_value"`
	Unit           *string   `gorm:"column:unit;size:20"`
	ReferenceRange *string   `gorm:"column:reference_range;size:500"`
	IsAbnormal     bool      `gorm:"column:is_abnormal;default:0"`
	Category       *string   `gorm:"column:category;size:50"`
	PatientID      int       `gorm:"column:patient_id;not null;index"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (DBExamResult) TableName() string { return "exam_results" }

// assessmentTables 导入后可能自动生成的评估表，撤销时也一并清理
var assessmentTables = []string{
	"fsp_risk_assessments_json",
	"ascvd_risk_assessments_json",
	"china_par_assessments_json",
	"risk_factors_summary_json",
	"assessment_reviews",
}

// rollbackTables 撤销时按顺序删除（叶→根），以 patient_id 或 id 过滤
var rollbackTables = []string{
	"exam_results",
	"health_examinations",
	"patient_modules",
	"fsp_risk_assessments_json",
	"ascvd_risk_assessments_json",
	"china_par_assessments_json",
	"risk_factors_summary_json",
	"assessment_reviews",
	"patients",
}
