package handler

import (
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type GuestLoginResp struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}

func issueGuestToken(visitorID string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-guest-secret" // 开发占位，生产请改成强随机
	}
	claims := jwt.MapClaims{
		"vid":  visitorID,
		"role": "guest",
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

func GuestLogin(c *gin.Context) {
	vid := c.GetString("visitor_id")
	if vid == "" {
		c.JSON(500, gin.H{"code": 500, "message": "visitor id missing"})
		return
	}
	token, err := issueGuestToken(vid)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "token error"})
		return
	}
	// 取 uuid 前缀做展示用户名
	short := vid
	if i := strings.IndexByte(vid, '-'); i > 0 {
		short = vid[:i]
	}
	c.JSON(200, GuestLoginResp{
		Token:    token,
		Username: "guest-" + short,
	})
}
