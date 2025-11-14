package main

import (
	"log"
	"time"

	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/pkg/config"
	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/pkg/middleware"
	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/pkg/mypubliclib/util"
	"github.com/gin-gonic/gin"

	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/handlers"
)

func main() {
	cfg, _ := config.Load()
	gin.SetMode(gin.ReleaseMode)

	// 初始化数据库连接并运行迁移（AutoMigrate 会自动创建表及索引）
	gormDB, err := config.Init(cfg)
	if err != nil {
		log.Fatal("db init error:", err)
	}

	// 创建 Gin 路由器，使用内置的恢复和自定义中间件
	r := gin.New()
	r.Use(gin.Recovery())       // 捕获 panic 并返回 500
	r.Use(util.Cors())          // CORS 跨域支持
	r.Use(middleware.Visitor()) // 为游客分配/识别 ID

	// 健康检查端点（用于负载均衡器和监控探测）
	r.GET("/api/v1/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "ts": time.Now().Unix()})
	})

	// 游客登录相关
	r.POST("/guest-login", handlers.GuestLogin(cfg))

	// 番茄钟计时及统计相关路由
	f := handlers.NewFocus(gormDB)

	r.POST("/api/v1/sessions/start", f.Start)    // 开始新的计时
	r.POST("/api/v1/sessions/pause", f.Pause)    // 暂停计时
	r.POST("/api/v1/sessions/resume", f.Resume)  // 恢复计时
	r.POST("/api/v1/sessions/finish", f.Finish)  // 完成计时
	r.POST("/api/v1/sessions/cancel", f.Cancel)  // 取消计时
	r.GET("/api/v1/sessions/current", f.Current) // 查询当前计时

	// 统计相关：今日/近7天/总计
	r.GET("/api/v1/stats/summary", f.Summary)

	// 成长事件：用于前端和宠物系统获取用户成长数据
	r.GET("/api/v1/events/growth/pull", f.GrowthPull) // 拉取未处理的成长事件，?limit=50
	r.POST("/api/v1/events/growth/ack", f.GrowthAck)  // 确认已处理的成长事件，body: {"last_id":123}

	//成就
	r.GET("/api/v1/achievements", f.Achievements)

	log.Println("listen on", cfg.Addr)
	if err := r.Run(cfg.Addr); err != nil {
		log.Fatal(err)
	}
}
