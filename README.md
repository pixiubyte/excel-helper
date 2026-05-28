# Demo Data Viewer

Go + Gin + GORM + SQLite + embed 单文件页面工具，用于扫描当前目录上一级的 `examinfo_*.xlsx` 和 `lis_*.xlsx`。

## 运行

```bash
cd /Users/hahaxi/code/python/yingqun/stroke-demo/.temp/demo-data/src
go mod tidy
go run . -data ..
```

默认地址：

```text
http://127.0.0.1:9188
```

## API

- `POST /api/scan`：后台扫描 Excel 并通过 GORM 写入 SQLite 索引
- `GET /api/scan/status`：扫描进度
- `GET /api/stats`：患者数、LIS 行数、异常数
- `GET /api/patients?page=1&page_size=50&q=26000000000`：分页列表
- `GET /api/patients/{EXAMINATION_ID}`：患者 examinfo + lis 明细
- `GET /api/stroke/batch?limit=50&offset=0`：给卒中项目拉取的批量 JSON
- `GET /api/stroke/patient/{EXAMINATION_ID}`：单患者 JSON

SQLite 文件默认生成在数据目录：

```text
.temp/demo-data/demo-data.db
```
