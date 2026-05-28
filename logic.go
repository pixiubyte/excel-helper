package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	patientInsertBatchSize = 50
	lisInsertBatchSize     = 500
)

func (a *App) initDB() error {
	return a.db.AutoMigrate(&Patient{}, &LISRecord{}, &LISSummary{}, &ScanMeta{})
}

func (a *App) scanAll() error {
	examFiles, err := filepath.Glob(filepath.Join(a.dataDir, "examinfo*.xlsx"))
	if err != nil {
		return err
	}
	lisFiles, err := filepath.Glob(filepath.Join(a.dataDir, "lis*.xlsx"))
	if err != nil {
		return err
	}
	sortStrings(examFiles)
	sortStrings(lisFiles)
	if len(examFiles) == 0 {
		return fmt.Errorf("未找到 examinfo*.xlsx: %s", a.dataDir)
	}
	if err := a.clearData(); err != nil {
		return err
	}
	a.setStatus(func(s *ScanStatus) {
		s.ExamFiles = len(examFiles)
		s.LisFiles = len(lisFiles)
		s.Message = "扫描 examinfo"
	})
	for _, path := range examFiles {
		if err := a.scanExaminfo(path); err != nil {
			return err
		}
	}
	a.setStatus(func(s *ScanStatus) { s.Message = "扫描 lis" })
	for _, path := range lisFiles {
		if err := a.scanLIS(path); err != nil {
			return err
		}
	}
	return a.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&ScanMeta{
		Key:   "scanned_at",
		Value: time.Now().Format(time.RFC3339),
	}).Error
}

func (a *App) clearData() error {
	return a.db.Transaction(func(tx *gorm.DB) error {
		session := tx.Session(&gorm.Session{AllowGlobalUpdate: true})
		if err := session.Delete(&Patient{}).Error; err != nil {
			return err
		}
		if err := session.Delete(&LISRecord{}).Error; err != nil {
			return err
		}
		if err := session.Delete(&LISSummary{}).Error; err != nil {
			return err
		}
		return session.Delete(&ScanMeta{}).Error
	})
}

func (a *App) scanExaminfo(path string) error {
	a.setStatus(func(s *ScanStatus) { s.CurrentFile = filepath.Base(path) })
	f, err := excelize.OpenFile(path, excelize.Options{RawCellValue: true})
	if err != nil {
		return err
	}
	defer f.Close()
	rows, err := f.Rows(f.GetSheetName(0))
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil
	}
	header, err := rows.Columns()
	if err != nil {
		return err
	}
	idx := headerIndex(header)
	rowNum := 1
	inserted := 0
	batch := make([]Patient, 0, patientInsertBatchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := a.db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(batch, patientInsertBatchSize).Error; err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}
	for rows.Next() {
		rowNum++
		cols, err := rows.Columns()
		if err != nil {
			return err
		}
		patient := buildPatient(header, cols, idx, filepath.Base(path), rowNum)
		if patient.ExamID == "" {
			continue
		}
		batch = append(batch, patient)
		inserted++
		if len(batch) >= patientInsertBatchSize {
			if err := flush(); err != nil {
				return err
			}
		}
		if inserted%5000 == 0 {
			a.addExamRows(5000)
		}
	}
	if err := flush(); err != nil {
		return err
	}
	a.addExamRows(int64(inserted % 5000))
	a.addExamRawRows(int64(rowNum - 1))
	return nil
}

func (a *App) scanLIS(path string) error {
	a.setStatus(func(s *ScanStatus) { s.CurrentFile = filepath.Base(path) })
	f, err := excelize.OpenFile(path, excelize.Options{RawCellValue: true})
	if err != nil {
		return err
	}
	defer f.Close()
	rows, err := f.Rows(f.GetSheetName(0))
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil
	}
	header, err := rows.Columns()
	if err != nil {
		return err
	}
	idx := headerIndex(header)

	type agg struct{ count, abnormal int }
	summaryBatch := map[string]*agg{}
	recordBatch := make([]LISRecord, 0, lisInsertBatchSize)
	base := filepath.Base(path)
	flushRecords := func() error {
		if len(recordBatch) == 0 {
			return nil
		}
		if err := a.db.CreateInBatches(recordBatch, lisInsertBatchSize).Error; err != nil {
			return err
		}
		recordBatch = recordBatch[:0]
		return nil
	}
	flushSummaries := func() error {
		if len(summaryBatch) == 0 {
			return nil
		}
		err := a.db.Transaction(func(tx *gorm.DB) error {
			for examID, v := range summaryBatch {
				var summary LISSummary
				result := tx.Where("exam_id = ?", examID).Find(&summary)
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected == 0 {
					summary = LISSummary{ExamID: examID, FirstFile: base}
				}
				if summary.ExamID == "" {
					summary.ExamID = examID
				}
				if summary.FirstFile == "" {
					summary.FirstFile = base
				}
				summary.LisCount += v.count
				summary.AbnormalCount += v.abnormal
				summary.LastFile = base
				if err := tx.Save(&summary).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		summaryBatch = map[string]*agg{}
		return nil
	}

	seen := 0
	rowNum := 1
	for rows.Next() {
		rowNum++
		cols, err := rows.Columns()
		if err != nil {
			return err
		}
		examID := getByIdx(cols, idx, "EXAMINATION_ID")
		if examID == "" {
			continue
		}
		recordBatch = append(recordBatch, buildLISRecord(header, cols, idx, base, rowNum))
		item := summaryBatch[examID]
		if item == nil {
			item = &agg{}
			summaryBatch[examID] = item
		}
		item.count++
		if isAbnormal(getByIdx(cols, idx, "YCTS")) {
			item.abnormal++
		}
		seen++
		if len(recordBatch) >= lisInsertBatchSize {
			if err := flushRecords(); err != nil {
				return err
			}
		}
		if seen%50000 == 0 {
			if err := flushRecords(); err != nil {
				return err
			}
			if err := flushSummaries(); err != nil {
				return err
			}
			a.addLisRows(50000)
		}
	}
	if err := flushRecords(); err != nil {
		return err
	}
	if err := flushSummaries(); err != nil {
		return err
	}
	a.addLisRows(int64(seen % 50000))
	a.addLisRawRows(int64(rowNum - 1))
	return nil
}

func (a *App) getPatientRaw(examID string) (map[string]any, error) {
	var patient Patient
	if err := a.db.First(&patient, "exam_id = ?", examID).Error; err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(patient.RawJSON), &m); err != nil {
		return nil, err
	}
	return m, nil
}

func buildPatient(header, cols []string, idx map[string]int, sourceFile string, rowNum int) Patient {
	examID := getByIdx(cols, idx, "EXAMINATION_ID")
	if examID == "" {
		return Patient{}
	}
	m := rowMap(header, cols)
	raw, _ := json.Marshal(m)
	return Patient{
		ExamID:         examID,
		YLJGDM:         getByIdx(cols, idx, "YLJGDM"),
		Zjhm:           getByIdx(cols, idx, "ZJHM"),
		Zjlx:           getByIdx(cols, idx, "ZJLX"),
		AppointId:      getByIdx(cols, idx, "APPOINT_ID"),
		PlanCode:       getByIdx(cols, idx, "PLAN_CODE"),
		Name:           nonEmpty(getByIdx(cols, idx, "PATIENT_NAME"), "患者_"+examID),
		Gender:         getByIdx(cols, idx, "XB"),
		BirthDate:      normalizeDate(getByIdx(cols, idx, "CSRQ")),
		ExamDate:       normalizeDate(getByIdx(cols, idx, "EXAMINATION_DATE")),
		DoctorId:       getByIdx(cols, idx, "DOCTOR_ID"),
		DoctorName:     getByIdx(cols, idx, "DOCTOR_NAME"),
		Zz:             getByIdx(cols, idx, "ZZ"),
		Zzmc:           getByIdx(cols, idx, "ZZMC"),
		Temperature:    getByIdx(cols, idx, "TEMPERATURE"),
		Ml:             getByIdx(cols, idx, "ML"),
		Hxpl:           getByIdx(cols, idx, "HXPL"),
		LeftSBP:        toFloat(getByIdx(cols, idx, "LEFT_SBP")),
		LeftDBP:        toFloat(getByIdx(cols, idx, "LEFT_DBP")),
		RightSBP:       toFloat(getByIdx(cols, idx, "RIGHT_SBP")),
		RightDBP:       toFloat(getByIdx(cols, idx, "RIGHT_DBP")),
		Height:         getByIdx(cols, idx, "HEIGHT"),
		Weight:         getByIdx(cols, idx, "WEIGHT"),
		Waistline:      toFloat(getByIdx(cols, idx, "WAISTLINE")),
		BMI:            toFloat(getByIdx(cols, idx, "BMI")),
		Hips:           getByIdx(cols, idx, "HIPS"),
		Whr:            getByIdx(cols, idx, "WHR"),
		Lnrrsgndm:      getByIdx(cols, idx, "LNRRSGNDM"),
		Lnrrsgnmc:      getByIdx(cols, idx, "LNRRSGNMC"),
		Lnrrzpf:        getByIdx(cols, idx, "LNRRZPF"),
		Zlztjczf:       getByIdx(cols, idx, "ZLZTJCZF"),
		Lnrjkzwpgdm:    getByIdx(cols, idx, "LNRJKZWPGDM"),
		Lnrjkzwpgmc:    getByIdx(cols, idx, "LNRJKZWPGMC"),
		Lnrzlnlpgdm:    getByIdx(cols, idx, "LNRZLNLPGDM"),
		Lnrzlnlpgmc:    getByIdx(cols, idx, "LNRZLNLPGMC"),
		Lnrqgztdm:      getByIdx(cols, idx, "LNRQGZTDM"),
		Lnrqgztmc:      getByIdx(cols, idx, "LNRQGZTMC"),
		Yypfjczf:       getByIdx(cols, idx, "YYPFJCZF"),
		Dlpldm:         getByIdx(cols, idx, "DLPLDM"),
		Dlplmc:         getByIdx(cols, idx, "DLPLMC"),
		Mcdlsj:         getByIdx(cols, idx, "MCDLSJ"),
		Jcdlsj:         getByIdx(cols, idx, "JCDLSJ"),
		Dlfs:           getByIdx(cols, idx, "DLFS"),
		Ysxgdm:         getByIdx(cols, idx, "YSXGDM"),
		Ysxgmc:         getByIdx(cols, idx, "YSXGMC"),
		Xyzkdm:         getByIdx(cols, idx, "XYZKDM"),
		Xyzkmc:         getByIdx(cols, idx, "XYZKMC"),
		Rxyl:           getByIdx(cols, idx, "RXYL"),
		Ksxynl:         getByIdx(cols, idx, "KSXYNL"),
		Jynl:           getByIdx(cols, idx, "JYNL"),
		Yjpldm:         getByIdx(cols, idx, "YJPLDM"),
		Yjplmc:         getByIdx(cols, idx, "YJPLMC"),
		Ryjl:           getByIdx(cols, idx, "RYJL"),
		Sfjj:           getByIdx(cols, idx, "SFJJ"),
		Jjnl:           getByIdx(cols, idx, "JJNL"),
		Ksyjnl:         getByIdx(cols, idx, "KSYJNL"),
		Sfzj:           getByIdx(cols, idx, "SFZJ"),
		Yjzldm:         getByIdx(cols, idx, "YJZLDM"),
		Yjzlmc:         getByIdx(cols, idx, "YJZLMC"),
		Kqkcdm:         getByIdx(cols, idx, "KQKCDM"),
		Kqkcmc:         getByIdx(cols, idx, "KQKCMC"),
		Kqcldm:         getByIdx(cols, idx, "KQCLDM"),
		Kqclmc:         getByIdx(cols, idx, "KQCLMC"),
		Kqclwz:         getByIdx(cols, idx, "KQCLWZ"),
		Kqclms:         getByIdx(cols, idx, "KQCLMS"),
		Kqybdm:         getByIdx(cols, idx, "KQYBDM"),
		Kqybmc:         getByIdx(cols, idx, "KQYBMC"),
		Slzy:           getByIdx(cols, idx, "SLZY"),
		Slyy:           getByIdx(cols, idx, "SLYY"),
		Jzslzy:         getByIdx(cols, idx, "JZSLZY"),
		Jzslyy:         getByIdx(cols, idx, "JZSLYY"),
		Slzydm:         getByIdx(cols, idx, "SLZYDM"),
		Slyydm:         getByIdx(cols, idx, "SLYYDM"),
		Ydycbz:         getByIdx(cols, idx, "YDYCBZ"),
		Ydycms:         getByIdx(cols, idx, "YDYCMS"),
		Tldm:           getByIdx(cols, idx, "TLDM"),
		Tlmc:           getByIdx(cols, idx, "TLMC"),
		Hdnldm:         getByIdx(cols, idx, "HDNLDM"),
		Hdnlmc:         getByIdx(cols, idx, "HDNLMC"),
		Pfdm:           getByIdx(cols, idx, "PFDM"),
		Pfmc:           getByIdx(cols, idx, "PFMC"),
		Gmdm:           getByIdx(cols, idx, "GMDM"),
		Gmmc:           getByIdx(cols, idx, "GMMC"),
		Lbjdm:          getByIdx(cols, idx, "LBJDM"),
		Lbjmc:          getByIdx(cols, idx, "LBJMC"),
		Ftzxbz:         getByIdx(cols, idx, "FTZXBZ"),
		Fhxybz:         getByIdx(cols, idx, "FHXYBZ"),
		Fhxyycms:       getByIdx(cols, idx, "FHXYYCMS"),
		Flydm:          getByIdx(cols, idx, "FLYDM"),
		Flymc:          getByIdx(cols, idx, "FLYMC"),
		Xtpl:           getByIdx(cols, idx, "XTPL"),
		Xldm:           getByIdx(cols, idx, "XLDM"),
		Xlmc:           getByIdx(cols, idx, "XLMC"),
		Xzzydm:         getByIdx(cols, idx, "XZZYDM"),
		Xzzyms:         getByIdx(cols, idx, "XZZYMS"),
		Fbytdm:         getByIdx(cols, idx, "FBYTDM"),
		Fbytms:         getByIdx(cols, idx, "FBYTMS"),
		Fbbkdm:         getByIdx(cols, idx, "FBBKDM"),
		Fbbkms:         getByIdx(cols, idx, "FBBKMS"),
		Fbgddm:         getByIdx(cols, idx, "FBGDDM"),
		Fbgdms:         getByIdx(cols, idx, "FBGDMS"),
		Fbpddm:         getByIdx(cols, idx, "FBPDDM"),
		Fbpdms:         getByIdx(cols, idx, "FBPDMS"),
		Fbydxzydm:      getByIdx(cols, idx, "FBYDXZYDM"),
		Fbydxzyms:      getByIdx(cols, idx, "FBYDXZYMS"),
		Xzszdm:         getByIdx(cols, idx, "XZSZDM"),
		Xzszmc:         getByIdx(cols, idx, "XZSZMC"),
		Zbdmbddm:       getByIdx(cols, idx, "ZBDMBDDM"),
		Zbdmbdmc:       getByIdx(cols, idx, "ZBDMBDMC"),
		Gmzzdm:         getByIdx(cols, idx, "GMZZDM"),
		Gmzzmc:         getByIdx(cols, idx, "GMZZMC"),
		Rxdm:           getByIdx(cols, idx, "RXDM"),
		Rxmc:           getByIdx(cols, idx, "RXMC"),
		Fkwydm:         getByIdx(cols, idx, "FKWYDM"),
		Fkwyms:         getByIdx(cols, idx, "FKWYMS"),
		Fkyddm:         getByIdx(cols, idx, "FKYDDM"),
		Fkydms:         getByIdx(cols, idx, "FKYDMS"),
		Fkgjdm:         getByIdx(cols, idx, "FKGJDM"),
		Fkgjms:         getByIdx(cols, idx, "FKGJMS"),
		Fkgtdm:         getByIdx(cols, idx, "FKGTDM"),
		Fkgtms:         getByIdx(cols, idx, "FKGTMS"),
		Fkfjdm:         getByIdx(cols, idx, "FKFJDM"),
		Fkfjms:         getByIdx(cols, idx, "FKFJMS"),
		Ctqt:           getByIdx(cols, idx, "CTQT"),
		Xdtbz:          getByIdx(cols, idx, "XDTBZ"),
		Xdtyc:          getByIdx(cols, idx, "XDTYC"),
		Xbxxpbz:        getByIdx(cols, idx, "XBXXPBZ"),
		Xbxxpyc:        getByIdx(cols, idx, "XBXXPYC"),
		Bcbz:           getByIdx(cols, idx, "BCBZ"),
		Bcyc:           getByIdx(cols, idx, "BCYC"),
		Gjtpbz:         getByIdx(cols, idx, "GJTPBZ"),
		Gjtpyc:         getByIdx(cols, idx, "GJTPYC"),
		Fzjcqt:         getByIdx(cols, idx, "FZJCQT"),
		Nxgjbdm:        getByIdx(cols, idx, "NXGJBDM"),
		Nxgjbmc:        getByIdx(cols, idx, "NXGJBMC"),
		Szjbdm:         getByIdx(cols, idx, "SZJBDM"),
		Szjbmc:         getByIdx(cols, idx, "SZJBMC"),
		Xzjbdm:         getByIdx(cols, idx, "XZJBDM"),
		Xzjbmc:         getByIdx(cols, idx, "XZJBMC"),
		Xgjbdm:         getByIdx(cols, idx, "XGJBDM"),
		Xgjbmc:         getByIdx(cols, idx, "XGJBMC"),
		Ybjbdm:         getByIdx(cols, idx, "YBJBDM"),
		Ybjbmc:         getByIdx(cols, idx, "YBJBMC"),
		Sjxtjbbz:       getByIdx(cols, idx, "SJXTJBBZ"),
		Sjxtjbqtms:     getByIdx(cols, idx, "SJXTJBQTMS"),
		Qtxtjbdm:       getByIdx(cols, idx, "QTXTJBDM"),
		Qtxtjbqtms:     getByIdx(cols, idx, "QTXTJBQTMS"),
		Jkpj:           getByIdx(cols, idx, "JKPJ"),
		Jkpjycqtms:     getByIdx(cols, idx, "JKPJYCQTMS"),
		Jkzd:           getByIdx(cols, idx, "JKZD"),
		Jkzdmc:         getByIdx(cols, idx, "JKZDMC"),
		Wxyskzdm:       getByIdx(cols, idx, "WXYSKZDM"),
		Wxyskzmc:       getByIdx(cols, idx, "WXYSKZMC"),
		Jtzmb:          getByIdx(cols, idx, "JTZMB"),
		Jyjzymdm:       getByIdx(cols, idx, "JYJZYMDM"),
		Jyjzymmc:       getByIdx(cols, idx, "JYJZYMMC"),
		Mj:             getByIdx(cols, idx, "MJ"),
		Xgbz:           getByIdx(cols, idx, "XGBZ"),
		Sjscsj:         getByIdx(cols, idx, "SJSCSJ"),
		Ylyl1:          getByIdx(cols, idx, "YLYL1"),
		Ylyl2:          getByIdx(cols, idx, "YLYL2"),
		WarFlag:        getByIdx(cols, idx, "WAR_FLAG"),
		WarNote:        getByIdx(cols, idx, "WAR_NOTE"),
		Jlgxsj:         getByIdx(cols, idx, "JLGXSJ"),
		Llbz:           getByIdx(cols, idx, "LLBZ"),
		ChkBz:          getByIdx(cols, idx, "CHK_BZ"),
		ExamSourceFile: sourceFile,
		ExamRowNum:     rowNum,
		RawJSON:        string(raw),
	}
}

func buildLISRecord(header, cols []string, idx map[string]int, sourceFile string, rowNum int) LISRecord {
	m := rowMap(header, cols)
	raw, _ := json.Marshal(m)
	return LISRecord{
		RecordID:      getByIdx(cols, idx, "RECORD_ID"),
		YLJGDM:        getByIdx(cols, idx, "YLJGDM"),
		ExamID:        getByIdx(cols, idx, "EXAMINATION_ID"),
		Jcxmdldm:      getByIdx(cols, idx, "JCXMDLDM"),
		Jcxmdlmc:      getByIdx(cols, idx, "JCXMDLMC"),
		Jcxmbm:        getByIdx(cols, idx, "JCXMBM"),
		Jcxmbmyb:      getByIdx(cols, idx, "JCXMBMYB"),
		Ybsfdm:        getByIdx(cols, idx, "YBSFDM"),
		Jczbmc:        getByIdx(cols, idx, "JCZBMC"),
		Jcff:          getByIdx(cols, idx, "JCFF"),
		Bgrq:          normalizeDate(getByIdx(cols, idx, "BGRQ")),
		Jczbjg:        getByIdx(cols, idx, "JCZBJG"),
		Jldw:          getByIdx(cols, idx, "JLDW"),
		Ckz:           getByIdx(cols, idx, "CKZ"),
		Ycts:          getByIdx(cols, idx, "YCTS"),
		Jcrgh:         getByIdx(cols, idx, "JCRGH"),
		Jcrxm:         getByIdx(cols, idx, "JCRXM"),
		Shrgh:         getByIdx(cols, idx, "SHRGH"),
		Shrxm:         getByIdx(cols, idx, "SHRXM"),
		Dyxh:          getByIdx(cols, idx, "DYXH"),
		Mj:            getByIdx(cols, idx, "MJ"),
		Xgbz:          getByIdx(cols, idx, "XGBZ"),
		Sjscsj:        getByIdx(cols, idx, "SJSCSJ"),
		Ylyl1:         getByIdx(cols, idx, "YLYL1"),
		Ylyl2:         getByIdx(cols, idx, "YLYL2"),
		WarFlag:       getByIdx(cols, idx, "WAR_FLAG"),
		WarNote:       getByIdx(cols, idx, "WAR_NOTE"),
		Jlgxsj:        getByIdx(cols, idx, "JLGXSJ"),
		Llbz:          getByIdx(cols, idx, "LLBZ"),
		ChkBz:         getByIdx(cols, idx, "CHK_BZ"),
		LisSourceFile: sourceFile,
		LisRowNum:     rowNum,
		RawJSON:       string(raw),
	}
}

func (a *App) findLisRows(examID string, limit int) ([]map[string]any, error) {
	var records []LISRecord
	if err := a.db.Where("exam_id = ?", examID).Order("id").Limit(limit).Find(&records).Error; err != nil {
		return nil, err
	}
	result := []map[string]any{}
	for _, record := range records {
		var raw map[string]any
		if err := json.Unmarshal([]byte(record.RawJSON), &raw); err != nil {
			return nil, err
		}
		result = append(result, raw)
	}
	return result, nil
}
