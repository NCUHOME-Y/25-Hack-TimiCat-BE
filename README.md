# 25-Hack-TimiCat-BE

后端骨架 (Go 1.25)

目录结构（主要部分）:

- cmd/TimiCat
- internal/{config,database,handlers,models}


## 快速开始
1. 在终端执行`cp .env.example .env`（按需修改端口）
2. 记得安装依赖，运行
```
go mod download
```
3. 打开docker 
4. 在终端运行`docker compose up -d` （起 Postgres + Adminer）
4. 运行程序`go run ./cmd/server/main.go`
5. 前端或 Apifox 访问：
   - POST `/guest-login`
   - POST `/api/v1/sessions/start、pause、resume、finish、cancel`
   - GET  `/api/v1/sessions/current`
   - GET  `/api/v1/stats/summary`
   - GET  `/api/v1/events/growth/pull?limit=50`
   - POST `/api/v1/events/growth/ack`

## 设计说明
- 使用 **GORM** 自动迁移，无需手写 SQL
- 统计数据采用 **Go 侧聚合**，逻辑简单
- 按 PRD 流程覆盖“开始/暂停/继续/结束/统计/成长事件”  


ps: 项目使用 `github.com/NCUHOME-Y/25-Hack-TimiCat-BE` 作为模块名。