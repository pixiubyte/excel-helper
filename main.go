package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	gmsql "gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

//go:embed templates/*
var embeddedFiles embed.FS

// App 持有全局依赖
type App struct {
	sqlite   *gorm.DB
	importer *Importer
}

func main() {
	mysqlDSN := flag.String("mysql",
		"stroke_user:stroke_pass@tcp(127.0.0.1:3306)/stroke_db?charset=utf8mb4&parseTime=True&loc=Local",
		"MySQL DSN（docker-compose 中的数据库）")
	sourceURL := flag.String("source", "http://127.0.0.1:9111", "9111 数据源地址")
	addr := flag.String("addr", ":9222", "监听地址")
	batchSize := flag.Int("batch", 100, "每次从数据源拉取的批量大小")
	sqliteDB := flag.String("sqlite", "import_tracker.db", "SQLite 追踪库路径")
	flag.Parse()

	// ── SQLite（必须）─────────────────────────────────────
	sdb, err := gorm.Open(sqlite.Open(*sqliteDB), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("SQLite 初始化失败: %v", err)
	}
	if err := sdb.AutoMigrate(&ImportBatch{}, &ImportBatchRecord{}, &ImportBatchLog{}); err != nil {
		log.Fatalf("SQLite migrate 失败: %v", err)
	}
	log.Printf("SQLite 已就绪: %s", *sqliteDB)

	// ── MySQL（可选）──────────────────────────────────────
	var mdb *gorm.DB
	mdb, err = gorm.Open(gmsql.Open(*mysqlDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Printf("⚠️  MySQL 连接失败 (%s)，页面仍可访问，导入功能不可用", maskDSN(*mysqlDSN))
		mdb = nil
	} else {
		sqlDB, _ := mdb.DB()
		sqlDB.SetMaxOpenConns(10)
		sqlDB.SetMaxIdleConns(5)
		log.Printf("MySQL 已就绪: %s", maskDSN(*mysqlDSN))
	}

	// ── 组装 ──────────────────────────────────────────────
	src := NewSourceClient(*sourceURL)
	imp := NewImporter(sdb, mdb, src, *batchSize)
	app := &App{sqlite: sdb, importer: imp}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	app.registerRoutes(r)

	log.Printf("🚀 服务启动 http://%s  (数据源 %s)", normalizeAddr(*addr), *sourceURL)
	if err := r.Run(*addr); err != nil {
		log.Fatal(err)
	}
}

func maskDSN(dsn string) string {
	// user:pass@tcp(...) → user:***@tcp(...)
	if i := strings.Index(dsn, ":"); i >= 0 {
		if j := strings.Index(dsn[i+1:], "@"); j >= 0 {
			return dsn[:i+1] + "***" + dsn[i+1+j:]
		}
	}
	return dsn
}

func normalizeAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return fmt.Sprintf("localhost%s", addr)
	}
	return addr
}
