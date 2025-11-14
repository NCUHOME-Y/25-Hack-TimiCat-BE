package middleware

import (
	"net/http"
	"strings"

	utils "github.com/NCUHOME-Y/25-Hack-TimiCat-BE/pkg/mypubliclib/util"
	"github.com/gin-gonic/gin"
)

// JWTAuth JWT鉴权
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否为空或格式不正确
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少Token或格式错误"})
			c.Abort()
			return
		}
		// 取出 token 字符串
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// 解析 Token
		claims, err := utils.ParseToken(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token无效或过期"})
			c.Abort()
			return
		}

		// 将用户信息放入请求的上下文
		c.Set("user_id", claims.UserID)
		c.Set("user_name", claims.UserName)
		c.Set("is_guest", claims.IsGuest)
		c.Next()
	}
}
