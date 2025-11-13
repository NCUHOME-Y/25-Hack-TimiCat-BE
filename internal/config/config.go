package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Env       string // 运行环境：dev 或 prod
	Addr      string // 服务绑定地址，例如 :3001
	JWTSecret string // JWT 签名密钥（用于游客身份验证）
	// Postgres 数据库配置
	PGUser string // 数据库用户名
	PGPass string // 数据库密码
	PGDB   string // 数据库名
	PGHost string // 数据库服务器地址
	PGPort string // 数据库服务器端口
}

// Load 从 .env 文件和环境变量读取配置
// 优先级：环境变量 > .env 文件 > 默认值
func Load() (*Config, error) {
	_ = godotenv.Load()

	c := &Config{
		Env:       get("ENV", "dev"),    // 默认开发环境
		Addr:      get("ADDR", ":3001"), // 默认监听 3001 端口
		JWTSecret: get("JWT_SECRET", "dev-guest-secret"),
		PGUser:    get("PGUSER", "app"),       // PostgreSQL 用户
		PGPass:    get("PGPASSWORD", "app"),   // PostgreSQL 密码
		PGDB:      get("PGDATABASE", "appdb"), // 数据库名
		PGHost:    get("PGHOST", "localhost"), // 数据库服务器地址
		PGPort:    get("PGPORT", "5432"),      // PostgreSQL 默认端口
	}
	_ = c // 为了提示器别报警
	return c, nil
}

func (c *Config) DSN() string {
	// GORM 的 PostgreSQL 驱动 DSN（数据源名称）格式
	// sslmode=disable 用于开发环境（生产环境应改为 require）
	// TimeZone 设置为上海时区，确保数据库时间与应用一致
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
		c.PGHost, c.PGUser, c.PGPass, c.PGDB, c.PGPort,
	)
}

// get 从环境变量获取值，如果为空则返回默认值
// 这样可以方便地处理可选配置，避免每个地方都写 if 判断
func get(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}
