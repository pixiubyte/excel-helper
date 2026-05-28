package main

import (
	"io/fs"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

func (a *App) setStatus(fn func(*ScanStatus)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	fn(a.status)
}

func (a *App) addExamRows(n int64) {
	if n <= 0 {
		return
	}
	a.mu.Lock()
	a.status.ExamRows += n
	a.mu.Unlock()
}

func (a *App) addLisRows(n int64) {
	if n <= 0 {
		return
	}
	a.mu.Lock()
	a.status.LisRows += n
	a.mu.Unlock()
}

func (a *App) addExamRawRows(n int64) {
	if n <= 0 {
		return
	}
	a.mu.Lock()
	a.status.ExamRawRows += n
	a.mu.Unlock()
}

func (a *App) addLisRawRows(n int64) {
	if n <= 0 {
		return
	}
	a.mu.Lock()
	a.status.LisRawRows += n
	a.mu.Unlock()
}

func rowMap(header, cols []string) map[string]any {
	m := make(map[string]any, len(header))
	for i, h := range header {
		if h == "" {
			continue
		}
		if i < len(cols) && cols[i] != "" {
			m[h] = cols[i]
		} else {
			m[h] = nil
		}
	}
	return m
}

func headerIndex(header []string) map[string]int {
	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[strings.TrimSpace(h)] = i
	}
	return idx
}

func getByIdx(cols []string, idx map[string]int, key string) string {
	i, ok := idx[key]
	if !ok || i >= len(cols) {
		return ""
	}
	return strings.TrimSpace(cols[i])
}

func isAbnormal(v string) bool {
	v = strings.TrimSpace(v)
	return v == "2" || v == "3" || v == "4" || v == "异常" || v == "↑" || v == "↓" || strings.EqualFold(v, "H") || strings.EqualFold(v, "L")
}

func normalizeDate(v string) string {
	v = strings.TrimSpace(v)
	if len(v) == 8 && allDigits(v) {
		return v[:4] + "-" + v[4:6] + "-" + v[6:8]
	}
	if len(v) >= 10 {
		return strings.ReplaceAll(v[:10], "/", "-")
	}
	return v
}

func allDigits(v string) bool {
	for _, r := range v {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func toFloat(v string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
	return f
}

func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func positiveInt(v string, fallback int) int {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

func clamp(n, min, max int) int {
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

func sortStrings(values []string) {
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
}

func embeddedStatic() http.FileSystem {
	sub, _ := fs.Sub(embeddedFiles, "templates")
	return http.FS(sub)
}
