package main

import (
	"embed"
	"flag"
	"log"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

//go:embed templates/*
var embeddedFiles embed.FS

func main() {
	defaultDir, _ := filepath.Abs("..")
	dataDir := flag.String("data", defaultDir, "包含 examinfo_*.xlsx 和 lis_*.xlsx 的目录")
	addr := flag.String("addr", ":9188", "监听地址")
	dbPath := flag.String("db", "", "SQLite 文件路径，默认在 data 目录下 demo-data.db")
	flag.Parse()

	if *dbPath == "" {
		*dbPath = filepath.Join(*dataDir, "demo-data.db")
	}
	db, err := gorm.Open(sqlite.Open(*dbPath+"?_busy_timeout=10000"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	app := &App{db: db, dataDir: *dataDir, status: &ScanStatus{Message: "未扫描"}}
	if err := app.initDB(); err != nil {
		log.Fatal(err)
	}

	router := gin.Default()
	app.registerRoutes(router)
	log.Printf("demo-data viewer listening on http://127.0.0.1%s data=%s db=%s", *addr, *dataDir, *dbPath)
	if err := router.Run(*addr); err != nil {
		log.Fatal(err)
	}
}
