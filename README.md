# export-by-outserver

Go 工具：从外网 9111 代理拉取 Excel 扫描数据，批量导入到内网 stroke-demo MySQL。

## 依赖

- Go 1.22+
- 内网可访问外网服务器 9111 端口
- 内网 MySQL（stroke-demo docker-compose）已启动

## 运行

```bash
cd build/export-by-outserver
go mod tidy
go run . \
  -mysql "stroke_user:StrokePass123@tcp(127.0.0.1:3306)/stroke_warning_system?charset=utf8mb4&parseTime=True&loc=Local" \
  -source "http://10.100.20.79:9111" \
  -addr ":9222" \
  -batch 100
```

打开浏览器访问 `http://127.0.0.1:9222`

## 参数说明

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-mysql` | `stroke_user:...@tcp(127.0.0.1:3306)/...` | MySQL DSN |
| `-source` | `http://127.0.0.1:9111` | 9111 代理地址 |
| `-addr` | `:9222` | 本工具 Web UI 端口 |
| `-batch` | `100` | 每批拉取患者数（最大 200）|

## 数据流

```
9111 /data/stroke/batch
    └── examinfo[] → patients + patient_modules(stroke)
                   → health_examinations
                   → exam_results (体征：身高/体重/血压/腰围/BMI)
    └── lis[]      → exam_results (LIS 检验项)
```

## Web UI 功能

- 查看数据源统计（患者数、LIS 行数、扫描状态）
- 查看 MySQL 已入库数量
- 一键开始 / 停止导入
- 支持从指定 offset 续传（断点续导）
- 实时进度 + 日志流

## 编译成单一可执行文件

```bash
go build -o export-by-outserver .
./export-by-outserver -source "http://10.100.20.79:9111" -mysql "..."
```
