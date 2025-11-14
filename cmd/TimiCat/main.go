package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/config"
	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/database"

	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/handlers"
)

// CORS 中间件：允许配置的源（本地前端开发用）
// 从环境变量读取允许列表，只有在列表内的请求才会获得 CORS 头
func cors() gin.HandlerFunc {
	allow := os.Getenv("ALLOW_ORIGINS")
	if allow == "" {
		// 默认允许常见本地开发地址（localhost/127.0.0.1 的 3000 与 5173 端口）
		allow = "http://localhost:3000,http://127.0.0.1:3000,http://localhost:5173,http://127.0.0.1:5173"
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		// 遍历允许列表，若请求来源在列表内则设置 CORS 头
		for _, a := range splitCSV(allow) {
			if origin == a {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
				c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				break
			}
		}
		// 对 OPTIONS 预检请求直接返回 204 No Content（浏览器跨域需要）
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	}
}

// splitCSV 将以逗号分隔的字符串分割成数组，并去除每个元素的前后空格
// 例如："http://localhost:3000, http://127.0.0.1:3000" 会被拆成两个元素
func splitCSV(s string) []string {
	out := []string{}
	t := ""
	for _, r := range s {
		if r == ',' {
			// 遇到逗号，把当前累积的字符串 trim 后加入结果
			if t != "" {
				out = append(out, trim(t))
			}
			t = ""
		} else {
			// 积累字符
			t += string(r)
		}
	}
	// 处理最后一个元素（最后面可能没有逗号）
	if t != "" {
		out = append(out, trim(t))
	}
	return out
}

// trim 从字符串的首尾删除空格和制表符
func trim(s string) string {
	// 删除前导空格和制表符
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	// 删除尾部空格和制表符
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

// visitor 中间件：为每个游客分配唯一 ID（存储在 cookie 中）
// 如果浏览器没有 tcid cookie，就生成一个新的 UUID 并设置，有效期一年
func visitor() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := c.Cookie("tcid"); err != nil {
			// Cookie 不存在或读取失败，为新游客签发 ID
			id := handlers.IssueVisitorID()
			// HttpOnly 防止 JS 读取；SameSite 防止跨站请求伪造
			// 开发环境用 Secure=false，上线改为 true（需要 HTTPS）
			c.SetCookie("tcid", id, 3600*24*365, "/", "", false, true)
		}
		c.Next()
	}
}

func main() {
	cfg, _ := config.Load()
	gin.SetMode(gin.ReleaseMode)

	// 初始化数据库连接并运行迁移（AutoMigrate 会自动创建表及索引）
	gormDB, err := database.InitGorm(cfg)
	if err != nil {
		log.Fatal("db init error:", err)
	}

	// 创建 Gin 路由器，使用内置的恢复和自定义中间件
	r := gin.New()
	r.Use(gin.Recovery()) // 捕获 panic 并返回 500
	r.Use(cors())         // CORS 跨域支持
	r.Use(visitor())      // 为游客分配/识别 ID

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
