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

ps: 项目使用 `github.com/NCUHOME-Y/25-Hack-TimiCat-BE` 作为模块名。
