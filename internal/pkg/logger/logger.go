package logger

import (
	"log"
	"os"
)

// 一个非常小的日志包装，提供项目中使用的方法
type Logger struct {
	std *log.Logger
}

// Init 创建一个简单的日志器，env 参数保留以兼容
func Init(env string) *Logger {
	l := log.New(os.Stdout, "", log.LstdFlags|log.Lmsgprefix)
	return &Logger{std: l}
}

func (l *Logger) Info(msg string, kvs ...interface{}) {
	l.std.Printf("INFO: %s %v", msg, kvs)
}

func (l *Logger) Debug(msg string, kvs ...interface{}) {
	l.std.Printf("DEBUG: %s %v", msg, kvs)
}

func (l *Logger) Error(msg string, kvs ...interface{}) {
	l.std.Printf("ERROR: %s %v", msg, kvs)
}

func (l *Logger) Fatal(msg string, kvs ...interface{}) {
	l.std.Printf("FATAL: %s %v", msg, kvs)
	os.Exit(1)
}

func (l *Logger) Sync() error { return nil }
