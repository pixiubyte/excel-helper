package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ─── 9111 数据源客户端 ───────────────────────────────────

type SourceClient struct {
	BaseURL    string
	httpClient *http.Client
}

func NewSourceClient(baseURL string) *SourceClient {
	return &SourceClient{
		BaseURL: baseURL,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *SourceClient) get(path string) ([]byte, error) {
	resp, err := c.httpClient.Get(c.BaseURL + path)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, path)
	}
	return io.ReadAll(resp.Body)
}

// SourceStats 对应 /data/stats
type SourceStats struct {
	Patients        int64  `json:"patients"`
	PatientsWithLis int64  `json:"patients_with_lis"`
	LisRows         int64  `json:"lis_rows"`
	AbnormalRows    int64  `json:"abnormal_rows"`
	DataDir         string `json:"data_dir"`
}

func (c *SourceClient) Stats() (*SourceStats, error) {
	b, err := c.get("/data/stats")
	if err != nil {
		return nil, err
	}
	var s SourceStats
	return &s, json.Unmarshal(b, &s)
}

// SourceScanStatus 对应 /data/scan/status
type SourceScanStatus struct {
	Running      bool      `json:"running"`
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at"`
	CurrentFile  string    `json:"current_file"`
	Message      string    `json:"message"`
	ExamFiles    int       `json:"exam_files"`
	LisFiles     int       `json:"lis_files"`
	ExamRows     int64     `json:"exam_rows"`
	LisRows      int64     `json:"lis_rows"`
	ErrorMessage string    `json:"error_message"`
}

func (c *SourceClient) ScanStatus() (*SourceScanStatus, error) {
	b, err := c.get("/data/scan/status")
	if err != nil {
		return nil, err
	}
	var s SourceScanStatus
	return &s, json.Unmarshal(b, &s)
}

// ─── 批量患者数据 ─────────────────────────────────────────

// SourceBatchResp 对应 /data/stroke/batch
type SourceBatchResp struct {
	Metadata struct {
		Limit         int `json:"limit"`
		Offset        int `json:"offset"`
		ExaminfoCount int `json:"examinfo_count"`
		LisCount      int `json:"lis_count"`
	} `json:"metadata"`
	Examinfo []json.RawMessage `json:"examinfo"`
	Lis      []SourceLisRecord `json:"lis"`
}

func (c *SourceClient) Batch(limit, offset int) (*SourceBatchResp, error) {
	path := fmt.Sprintf("/data/stroke/batch?limit=%d&offset=%d", limit, offset)
	b, err := c.get(path)
	if err != nil {
		return nil, err
	}
	var resp SourceBatchResp
	return &resp, json.Unmarshal(b, &resp)
}

// ─── 患者列表（预览）─────────────────────────────────────

// SourcePatientList 对应 /data/patients
type SourcePatientList struct {
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
	Total    int64                 `json:"total"`
	Items    []SourcePatientItem   `json:"items"`
}

type SourcePatientItem struct {
	ExamID         string  `json:"exam_id"`
	YLJGDM         string  `json:"yljgdm"`
	Name           string  `json:"name"`
	Gender         string  `json:"gender"`
	BirthDate      string  `json:"birth_date"`
	ExamDate       string  `json:"exam_date"`
	LeftSBP        float64 `json:"left_sbp"`
	LeftDBP        float64 `json:"left_dbp"`
	RightSBP       float64 `json:"right_sbp"`
	RightDBP       float64 `json:"right_dbp"`
	Waistline      float64 `json:"waistline"`
	BMI            float64 `json:"bmi"`
	LisCount       int     `json:"lis_count"`
	AbnormalCount  int     `json:"abnormal_count"`
	ExamSourceFile string  `json:"exam_source_file"`
}

func (c *SourceClient) Patients(page, pageSize int, q string) (*SourcePatientList, error) {
	path := fmt.Sprintf("/data/patients?page=%d&page_size=%d", page, pageSize)
	if q != "" {
		path += "&q=" + q
	}
	b, err := c.get(path)
	if err != nil {
		return nil, err
	}
	var resp SourcePatientList
	return &resp, json.Unmarshal(b, &resp)
}

// ─── LIS / 体检原始字段 ──────────────────────────────────

// SourceLisRecord 对应 LIS 字段
type SourceLisRecord struct {
	ExamID   string `json:"EXAMINATION_ID"`
	Jcxmbm   string `json:"JCXMBM"`   // 项目编码
	Jczbmc   string `json:"JCZBMC"`   // 项目名称
	Jczbjg   string `json:"JCZBJG"`   // 结果值
	Jldw     string `json:"JLDW"`     // 单位
	Ckz      string `json:"CKZ"`      // 参考范围
	Ycts     string `json:"YCTS"`     // 异常提示
	Jcxmdlmc string `json:"JCXMDLMC"` // 套餐分类
}

// SourceExaminfoRecord 体检记录字段（合并英文/中文列名）
type SourceExaminfoRecord struct {
	// 英文字段（Go SQLite 结构化字段）
	ExamID    string  `json:"EXAMINATION_ID"`
	Name      string  `json:"PATIENT_NAME"`
	Gender    string  `json:"PATIENT_GENDER"`
	BirthDate string  `json:"PATIENT_BIRTHDAY"`
	ExamDate  string  `json:"EXAMINATION_DATE"`
	LeftSBP   float64 `json:"left_sbp"`
	LeftDBP   float64 `json:"left_dbp"`
	RightSBP  float64 `json:"right_sbp"`
	RightDBP  float64 `json:"right_dbp"`
	Waistline float64 `json:"waistline"`
	BMI       float64 `json:"bmi"`
	Height    string  `json:"height"`
	Weight    string  `json:"weight"`

	// 中文列名（原始 Excel 字段）
	ExamIDCN    string `json:"体检编号"`
	NameCN      string `json:"姓名"`
	GenderCN    string `json:"性别"`
	BirthDateCN string `json:"出生日期"`
	ExamDateCN  string `json:"体检日期"`
	HeightCN    string `json:"身高(cm)"`
	WeightCN    string `json:"体重(kg)"`
	WaistCN     string `json:"腰围(cm)"`
	BMICN       string `json:"BMI(kg/m²)"`
	SBPCN       string `json:"收缩压(mmHg)"`
	DBPCN       string `json:"舒张压(mmHg)"`
	LeftSBPCN   string `json:"左臂收缩压(mmHg)"`
	LeftDBPCN   string `json:"左臂舒张压(mmHg)"`
	RightSBPCN  string `json:"右臂收缩压(mmHg)"`
	RightDBPCN  string `json:"右臂舒张压(mmHg)"`
}

// Normalize 统一字段：中文字段优先
func (r *SourceExaminfoRecord) Normalize() {
	if r.ExamID == "" {
		r.ExamID = r.ExamIDCN
	}
	if r.Name == "" {
		r.Name = r.NameCN
	}
	if r.Gender == "" {
		r.Gender = r.GenderCN
	}
	if r.BirthDate == "" {
		r.BirthDate = r.BirthDateCN
	}
	if r.ExamDate == "" {
		r.ExamDate = r.ExamDateCN
	}
}

// Health 检测数据源连通性
func (c *SourceClient) Health() bool {
	b, err := c.get("/data/health")
	return err == nil && len(b) > 0
}
