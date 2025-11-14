package handlers

import (
	"time"

	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/pkg/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// IssueVisitorID 生成游客 cookie 用的 uuid
func IssueVisitorID() string { return uuid.NewString() }

// 简单签发 JWT（给前端存 localStorage 用）
func signGuestToken(secret, visitorID string) (string, error) {
	claims := jwt.MapClaims{
		"vid":  visitorID,
		"role": "guest",
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

// POST /guest-login
// 返回 token（不返回 username）
func GuestLogin(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		vid, _ := c.Cookie("tcid") // 有则复用
		if vid == "" {
			vid = IssueVisitorID()
			// SameSite=Lax + HttpOnly；上线到 HTTPS 后可把 secure=true
			c.SetCookie("tcid", vid, 3600*24*365, "/", "", false, true)
		}
		token, err := signGuestToken(cfg.JWTSecret, vid)
		if err != nil {
			c.JSON(500, gin.H{"code": 500, "message": "token error"})
			return
		}
		c.JSON(200, gin.H{"token": token})
	}
}

// GET /me  仅用于校验/拿 visitorId（不返回 username）
func Me() gin.HandlerFunc {
	return func(c *gin.Context) {
		vid, err := c.Cookie("tcid")
		if err != nil || vid == "" {
			c.JSON(401, gin.H{"code": 401, "message": "unauthorized"})
			return
		}
		c.JSON(200, gin.H{"visitorId": vid})
	}
}
