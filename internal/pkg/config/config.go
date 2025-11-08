package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Env  string
	Addr string
}

// Load 读取 .env（如果存在）并加载环境变量
func Load() (*Config, error) {
	_ = godotenv.Load()

	env := os.Getenv("ENV")
	if env == "" {
		env = "dev"
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":3001"
	}
	return &Config{Env: env, Addr: addr}, nil
}
