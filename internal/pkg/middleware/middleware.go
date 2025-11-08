package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/pkg/logger"
	"github.com/google/uuid"
)

// RequestID 在context和响应头中加入请求 ID
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = uuid.New().String()
		}
		ctx := context.WithValue(r.Context(), "request_id", id)
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CORS 简单的中间件，改了一次变得不简单了:(
func CORS(next http.Handler) http.Handler {
	allow := os.Getenv("ALLOW_ORIGINS")
	if allow == "" {
		allow = "http://localhost:3000,http://127.0.0.1:3000,http://localhost:5173,http://127.0.0.1:5173"
	}
	whitelist := strings.Split(allow, ",")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		for _, o := range whitelist {
			if origin == strings.TrimSpace(o) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				break
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Recovery 返回一个从 panic 中恢复中的间件并打印日志
func Recovery(log *logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered", rec)
					http.Error(w, "internal error", http.StatusInternalServerError)
				}
			}()
			start := time.Now()
			next.ServeHTTP(w, r)
			// 基础日志记录
			log.Debug("handled request", map[string]interface{}{"method": r.Method, "path": r.URL.Path, "dur": time.Since(start)})
		})
	}
}
