# 25-Hack-TimiCat-BE

后端骨架 (Go 1.25)

目录结构（主要部分）:

- cmd/app
- internal/app/{handler,service,repository,model}
- internal/pkg/{config,logger,err,middleware}
- migrations
- test/integration

使用

1. 复制 `.env.example` 为 `.env` 并根据需要修改
2. 运行开发模式:

```powershell
make run
```

3. 运行测试:

```powershell
make test
```
4. 数据库相关操作请看 (先起数据库)25-Hack-TimiCat-BE\docker-compose.yml

5.记得安装依赖

```
go mod download
```

6.环境
```
# 环境：把示例复制成真实 .env
# 确保里面是：
#   APP_PORT=3001
#   DB_DSN=postgres://app:app@localhost:5432/appdb?sslmode=disable
copy .env.example .env
```

ps: 项目使用 `github.com/NCUHOME-Y/25-Hack-TimiCat-BE` 作为模块名。

# Windows PowerShell
docker compose up -d           # 起 Postgres + Adminer
$env:DB_DSN="postgres://app:app@localhost:5432/appdb?sslmode=disable"
$env:MIN_SESSION_SEC="60"      # 开发可用 5；生产建议 60
go run ./cmd/app
