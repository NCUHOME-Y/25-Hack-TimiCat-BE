package config

import (
	"time"

	"gorm.io/gorm"
)

var DB *gorm.DB

// AppConfig 配置结构体
type AppConfig struct {
	ServerPort int
	JWTSecret  string
	JWTExpire  time.Duration
	DBUser     string
	DBPassword string
	DBHost     string
	DBPort     int
	DBName     string
}

var Cfg = AppConfig{
	ServerPort: 8080,
	JWTSecret:  "my_super_secret_key",
	JWTExpire:  1024 * time.Hour, //Token 有效时间
	DBUser:     "root",
	DBPassword: "20070714",
	DBHost:     "127.0.0.1",
	DBPort:     3306,
	DBName:     "timicat",
}
