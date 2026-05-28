package main

import (
	"sync"
	"time"

	"gorm.io/gorm"
)

type App struct {
	db      *gorm.DB
	dataDir string
	status  *ScanStatus
	mu      sync.Mutex
}

type Patient struct {
	ExamID         string  `gorm:"primaryKey;column:exam_id"`
	YLJGDM         string  `gorm:"column:yljgdm;index"`
	Zjhm           string  `gorm:"column:zjhm"`
	Zjlx           string  `gorm:"column:zjlx"`
	AppointId      string  `gorm:"column:appoint_id"`
	PlanCode       string  `gorm:"column:plan_code"`
	Name           string  `gorm:"column:name;index"`
	Gender         string  `gorm:"column:gender"`
	BirthDate      string  `gorm:"column:birth_date"`
	ExamDate       string  `gorm:"column:exam_date;index"`
	DoctorId       string  `gorm:"column:doctor_id"`
	DoctorName     string  `gorm:"column:doctor_name"`
	Zz             string  `gorm:"column:zz"`
	Zzmc           string  `gorm:"column:zzmc"`
	Temperature    string  `gorm:"column:temperature"`
	Ml             string  `gorm:"column:ml"`
	Hxpl           string  `gorm:"column:hxpl"`
	LeftSBP        float64 `gorm:"column:left_sbp"`
	LeftDBP        float64 `gorm:"column:left_dbp"`
	RightSBP       float64 `gorm:"column:right_sbp"`
	RightDBP       float64 `gorm:"column:right_dbp"`
	Height         string  `gorm:"column:height"`
	Weight         string  `gorm:"column:weight"`
	Waistline      float64 `gorm:"column:waistline"`
	BMI            float64 `gorm:"column:bmi"`
	Hips           string  `gorm:"column:hips"`
	Whr            string  `gorm:"column:whr"`
	Lnrrsgndm      string  `gorm:"column:lnrrsgndm"`
	Lnrrsgnmc      string  `gorm:"column:lnrrsgnmc"`
	Lnrrzpf        string  `gorm:"column:lnrrzpf"`
	Zlztjczf       string  `gorm:"column:zlztjczf"`
	Lnrjkzwpgdm    string  `gorm:"column:lnrjkzwpgdm"`
	Lnrjkzwpgmc    string  `gorm:"column:lnrjkzwpgmc"`
	Lnrzlnlpgdm    string  `gorm:"column:lnrzlnlpgdm"`
	Lnrzlnlpgmc    string  `gorm:"column:lnrzlnlpgmc"`
	Lnrqgztdm      string  `gorm:"column:lnrqgztdm"`
	Lnrqgztmc      string  `gorm:"column:lnrqgztmc"`
	Yypfjczf       string  `gorm:"column:yypfjczf"`
	Dlpldm         string  `gorm:"column:dlpldm"`
	Dlplmc         string  `gorm:"column:dlplmc"`
	Mcdlsj         string  `gorm:"column:mcdlsj"`
	Jcdlsj         string  `gorm:"column:jcdlsj"`
	Dlfs           string  `gorm:"column:dlfs"`
	Ysxgdm         string  `gorm:"column:ysxgdm"`
	Ysxgmc         string  `gorm:"column:ysxgmc"`
	Xyzkdm         string  `gorm:"column:xyzkdm"`
	Xyzkmc         string  `gorm:"column:xyzkmc"`
	Rxyl           string  `gorm:"column:rxyl"`
	Ksxynl         string  `gorm:"column:ksxynl"`
	Jynl           string  `gorm:"column:jynl"`
	Yjpldm         string  `gorm:"column:yjpldm"`
	Yjplmc         string  `gorm:"column:yjplmc"`
	Ryjl           string  `gorm:"column:ryjl"`
	Sfjj           string  `gorm:"column:sfjj"`
	Jjnl           string  `gorm:"column:jjnl"`
	Ksyjnl         string  `gorm:"column:ksyjnl"`
	Sfzj           string  `gorm:"column:sfzj"`
	Yjzldm         string  `gorm:"column:yjzldm"`
	Yjzlmc         string  `gorm:"column:yjzlmc"`
	Kqkcdm         string  `gorm:"column:kqkcdm"`
	Kqkcmc         string  `gorm:"column:kqkcmc"`
	Kqcldm         string  `gorm:"column:kqcldm"`
	Kqclmc         string  `gorm:"column:kqclmc"`
	Kqclwz         string  `gorm:"column:kqclwz"`
	Kqclms         string  `gorm:"column:kqclms"`
	Kqybdm         string  `gorm:"column:kqybdm"`
	Kqybmc         string  `gorm:"column:kqybmc"`
	Slzy           string  `gorm:"column:slzy"`
	Slyy           string  `gorm:"column:slyy"`
	Jzslzy         string  `gorm:"column:jzslzy"`
	Jzslyy         string  `gorm:"column:jzslyy"`
	Slzydm         string  `gorm:"column:slzydm"`
	Slyydm         string  `gorm:"column:slyydm"`
	Ydycbz         string  `gorm:"column:ydycbz"`
	Ydycms         string  `gorm:"column:ydycms"`
	Tldm           string  `gorm:"column:tldm"`
	Tlmc           string  `gorm:"column:tlmc"`
	Hdnldm         string  `gorm:"column:hdnldm"`
	Hdnlmc         string  `gorm:"column:hdnlmc"`
	Pfdm           string  `gorm:"column:pfdm"`
	Pfmc           string  `gorm:"column:pfmc"`
	Gmdm           string  `gorm:"column:gmdm"`
	Gmmc           string  `gorm:"column:gmmc"`
	Lbjdm          string  `gorm:"column:lbjdm"`
	Lbjmc          string  `gorm:"column:lbjmc"`
	Ftzxbz         string  `gorm:"column:ftzxbz"`
	Fhxybz         string  `gorm:"column:fhxybz"`
	Fhxyycms       string  `gorm:"column:fhxyycms"`
	Flydm          string  `gorm:"column:flydm"`
	Flymc          string  `gorm:"column:flymc"`
	Xtpl           string  `gorm:"column:xtpl"`
	Xldm           string  `gorm:"column:xldm"`
	Xlmc           string  `gorm:"column:xlmc"`
	Xzzydm         string  `gorm:"column:xzzydm"`
	Xzzyms         string  `gorm:"column:xzzyms"`
	Fbytdm         string  `gorm:"column:fbytdm"`
	Fbytms         string  `gorm:"column:fbytms"`
	Fbbkdm         string  `gorm:"column:fbbkdm"`
	Fbbkms         string  `gorm:"column:fbbkms"`
	Fbgddm         string  `gorm:"column:fbgddm"`
	Fbgdms         string  `gorm:"column:fbgdms"`
	Fbpddm         string  `gorm:"column:fbpddm"`
	Fbpdms         string  `gorm:"column:fbpdms"`
	Fbydxzydm      string  `gorm:"column:fbydxzydm"`
	Fbydxzyms      string  `gorm:"column:fbydxzyms"`
	Xzszdm         string  `gorm:"column:xzszdm"`
	Xzszmc         string  `gorm:"column:xzszmc"`
	Zbdmbddm       string  `gorm:"column:zbdmbddm"`
	Zbdmbdmc       string  `gorm:"column:zbdmbdmc"`
	Gmzzdm         string  `gorm:"column:gmzzdm"`
	Gmzzmc         string  `gorm:"column:gmzzmc"`
	Rxdm           string  `gorm:"column:rxdm"`
	Rxmc           string  `gorm:"column:rxmc"`
	Fkwydm         string  `gorm:"column:fkwydm"`
	Fkwyms         string  `gorm:"column:fkwyms"`
	Fkyddm         string  `gorm:"column:fkyddm"`
	Fkydms         string  `gorm:"column:fkydms"`
	Fkgjdm         string  `gorm:"column:fkgjdm"`
	Fkgjms         string  `gorm:"column:fkgjms"`
	Fkgtdm         string  `gorm:"column:fkgtdm"`
	Fkgtms         string  `gorm:"column:fkgtms"`
	Fkfjdm         string  `gorm:"column:fkfjdm"`
	Fkfjms         string  `gorm:"column:fkfjms"`
	Ctqt           string  `gorm:"column:ctqt"`
	Xdtbz          string  `gorm:"column:xdtbz"`
	Xdtyc          string  `gorm:"column:xdtyc"`
	Xbxxpbz        string  `gorm:"column:xbxxpbz"`
	Xbxxpyc        string  `gorm:"column:xbxxpyc"`
	Bcbz           string  `gorm:"column:bcbz"`
	Bcyc           string  `gorm:"column:bcyc"`
	Gjtpbz         string  `gorm:"column:gjtpbz"`
	Gjtpyc         string  `gorm:"column:gjtpyc"`
	Fzjcqt         string  `gorm:"column:fzjcqt"`
	Nxgjbdm        string  `gorm:"column:nxgjbdm"`
	Nxgjbmc        string  `gorm:"column:nxgjbmc"`
	Szjbdm         string  `gorm:"column:szjbdm"`
	Szjbmc         string  `gorm:"column:szjbmc"`
	Xzjbdm         string  `gorm:"column:xzjbdm"`
	Xzjbmc         string  `gorm:"column:xzjbmc"`
	Xgjbdm         string  `gorm:"column:xgjbdm"`
	Xgjbmc         string  `gorm:"column:xgjbmc"`
	Ybjbdm         string  `gorm:"column:ybjbdm"`
	Ybjbmc         string  `gorm:"column:ybjbmc"`
	Sjxtjbbz       string  `gorm:"column:sjxtjbbz"`
	Sjxtjbqtms     string  `gorm:"column:sjxtjbqtms"`
	Qtxtjbdm       string  `gorm:"column:qtxtjbdm"`
	Qtxtjbqtms     string  `gorm:"column:qtxtjbqtms"`
	Jkpj           string  `gorm:"column:jkpj"`
	Jkpjycqtms     string  `gorm:"column:jkpjycqtms"`
	Jkzd           string  `gorm:"column:jkzd"`
	Jkzdmc         string  `gorm:"column:jkzdmc"`
	Wxyskzdm       string  `gorm:"column:wxyskzdm"`
	Wxyskzmc       string  `gorm:"column:wxyskzmc"`
	Jtzmb          string  `gorm:"column:jtzmb"`
	Jyjzymdm       string  `gorm:"column:jyjzymdm"`
	Jyjzymmc       string  `gorm:"column:jyjzymmc"`
	Mj             string  `gorm:"column:mj"`
	Xgbz           string  `gorm:"column:xgbz"`
	Sjscsj         string  `gorm:"column:sjscsj"`
	Ylyl1          string  `gorm:"column:ylyl1"`
	Ylyl2          string  `gorm:"column:ylyl2"`
	WarFlag        string  `gorm:"column:war_flag"`
	WarNote        string  `gorm:"column:war_note"`
	Jlgxsj         string  `gorm:"column:jlgxsj"`
	Llbz           string  `gorm:"column:llbz"`
	ChkBz          string  `gorm:"column:chk_bz"`
	ExamSourceFile string  `gorm:"column:exam_source_file"`
	ExamRowNum     int     `gorm:"column:exam_row_num"`
	RawJSON        string  `gorm:"column:raw_json"`
}

func (Patient) TableName() string { return "patients" }

type LISRecord struct {
	ID            uint   `gorm:"primaryKey;column:id" json:"-"`
	RecordID      string `gorm:"column:record_id;index" json:"RECORD_ID,omitempty"`
	YLJGDM        string `gorm:"column:yljgdm" json:"YLJGDM,omitempty"`
	ExamID        string `gorm:"column:exam_id;index" json:"EXAMINATION_ID,omitempty"`
	Jcxmdldm      string `gorm:"column:jcxmdldm" json:"JCXMDLDM,omitempty"`
	Jcxmdlmc      string `gorm:"column:jcxmdlmc" json:"JCXMDLMC,omitempty"`
	Jcxmbm        string `gorm:"column:jcxmbm" json:"JCXMBM,omitempty"`
	Jcxmbmyb      string `gorm:"column:jcxmbmyb" json:"JCXMBMYB,omitempty"`
	Ybsfdm        string `gorm:"column:ybsfdm" json:"YBSFDM,omitempty"`
	Jczbmc        string `gorm:"column:jczbmc" json:"JCZBMC,omitempty"`
	Jcff          string `gorm:"column:jcff" json:"JCFF,omitempty"`
	Bgrq          string `gorm:"column:bgrq" json:"BGRQ,omitempty"`
	Jczbjg        string `gorm:"column:jczbjg" json:"JCZBJG,omitempty"`
	Jldw          string `gorm:"column:jldw" json:"JLDW,omitempty"`
	Ckz           string `gorm:"column:ckz" json:"CKZ,omitempty"`
	Ycts          string `gorm:"column:ycts;index" json:"YCTS,omitempty"`
	Jcrgh         string `gorm:"column:jcrgh" json:"JCRGH,omitempty"`
	Jcrxm         string `gorm:"column:jcrxm" json:"JCRXM,omitempty"`
	Shrgh         string `gorm:"column:shrgh" json:"SHRGH,omitempty"`
	Shrxm         string `gorm:"column:shrxm" json:"SHRXM,omitempty"`
	Dyxh          string `gorm:"column:dyxh" json:"DYXH,omitempty"`
	Mj            string `gorm:"column:mj" json:"MJ,omitempty"`
	Xgbz          string `gorm:"column:xgbz" json:"XGBZ,omitempty"`
	Sjscsj        string `gorm:"column:sjscsj" json:"SJSCSJ,omitempty"`
	Ylyl1         string `gorm:"column:ylyl1" json:"YLYL1,omitempty"`
	Ylyl2         string `gorm:"column:ylyl2" json:"YLYL2,omitempty"`
	WarFlag       string `gorm:"column:war_flag" json:"WAR_FLAG,omitempty"`
	WarNote       string `gorm:"column:war_note" json:"WAR_NOTE,omitempty"`
	Jlgxsj        string `gorm:"column:jlgxsj" json:"JLGXSJ,omitempty"`
	Llbz          string `gorm:"column:llbz" json:"LLBZ,omitempty"`
	ChkBz         string `gorm:"column:chk_bz" json:"CHK_BZ,omitempty"`
	LisSourceFile string `gorm:"column:lis_source_file" json:"LIS_SOURCE_FILE,omitempty"`
	LisRowNum     int    `gorm:"column:lis_row_num" json:"LIS_ROW_NUM,omitempty"`
	RawJSON       string `gorm:"column:raw_json" json:"-"`
}

func (LISRecord) TableName() string { return "lis_records" }

type LISSummary struct {
	ExamID        string `gorm:"primaryKey;column:exam_id"`
	LisCount      int    `gorm:"column:lis_count;index"`
	AbnormalCount int    `gorm:"column:abnormal_count"`
	FirstFile     string `gorm:"column:first_file"`
	LastFile      string `gorm:"column:last_file"`
}

func (LISSummary) TableName() string { return "lis_summary" }

type ScanMeta struct {
	Key   string `gorm:"primaryKey;column:key"`
	Value string `gorm:"column:value"`
}

func (ScanMeta) TableName() string { return "scan_meta" }

type ScanStatus struct {
	Running      bool      `json:"running"`
	StartedAt    time.Time `json:"started_at,omitempty"`
	FinishedAt   time.Time `json:"finished_at,omitempty"`
	CurrentFile  string    `json:"current_file,omitempty"`
	Message      string    `json:"message"`
	ExamFiles    int       `json:"exam_files"`
	LisFiles     int       `json:"lis_files"`
	ExamRows     int64     `json:"exam_rows"`
	ExamRawRows  int64     `json:"exam_raw_rows"`
	LisRows      int64     `json:"lis_rows"`
	LisRawRows   int64     `json:"lis_raw_rows"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

type PatientListItem struct {
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
	FirstLisFile   string  `json:"first_lis_file"`
	LastLisFile    string  `json:"last_lis_file"`
}
