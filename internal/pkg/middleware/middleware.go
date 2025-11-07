package middleware

import (
	"context"
	"net/http"
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

// CORS 简单的中间件
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Request-Id")
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
