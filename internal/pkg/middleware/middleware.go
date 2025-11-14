package middleware

import (
	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/handlers"
	"github.com/gin-gonic/gin"
)

// Visitor  中间件：为每个游客分配唯一 ID（存储在 cookie 中）
// 如果浏览器没有 tcid cookie，就生成一个新的 UUID 并设置，有效期一年
func Visitor() gin.HandlerFunc {
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
