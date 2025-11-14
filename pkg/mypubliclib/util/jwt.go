package utils

import (
	"errors"
	"strings"
	"time"

	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/pkg/config"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   uint64 `json:"user_id"`
	UserName string `json:"user_name"`
	IsGuest  bool   `json:"is_guest"`
	jwt.RegisteredClaims
}

// GenerateToken 生成Token
func GenerateToken(userID uint64, username string, isGuest bool) (string, error) {
	expirationTime := time.Now().Add(config.Cfg.JWTExpire)
	claims := &Claims{
		UserID:   userID,
		UserName: username,
		IsGuest:  isGuest,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime), // Token过期时间
			IssuedAt:  jwt.NewNumericDate(time.Now()),     // Token签发时间
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims) // 生成 Token（指定签名算法为 HS256，对称加密）
	return token.SignedString([]byte(config.Cfg.JWTSecret))    // 用密钥签名 Token，生成最终字符串并返回
}

// ParseToken 验证 Token 的签名并提取自定义声明
func ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Cfg.JWTSecret), nil
	}) // 解析Token
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("无效的token")
	}
	return claims, nil
}

// ExactToken 从请求头中提取token字符串
func ExactToken(authHeader string) string {
	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 && parts[1] == "Bearer" {
		return parts[1]
	}
	return ""
}
