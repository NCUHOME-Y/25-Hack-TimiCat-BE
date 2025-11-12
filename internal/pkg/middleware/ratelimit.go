package middleware

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

type keyFunc = func(r *http.Request) string

// 每个 key 每秒 5 次，瞬时突发 10 次；可根据需要调整
func RateLimit(next http.Handler, kf keyFunc) http.Handler {
	limiter := struct {
		mu sync.Mutex
		m  map[string]*rate.Limiter
	}{m: map[string]*rate.Limiter{}}

	get := func(k string) *rate.Limiter {
		limiter.mu.Lock()
		defer limiter.mu.Unlock()
		if l, ok := limiter.m[k]; ok {
			return l
		}
		l := rate.NewLimiter(5, 10)
		limiter.m[k] = l
		return l
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		k := kf(r)
		if k == "" {
			host, _, _ := net.SplitHostPort(r.RemoteAddr)
			k = host
		}
		if !get(k).Allow() {
			w.Header().Set("稍后重试", "1")
			http.Error(w, "请求频繁", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// 从 cookie 获取访客键
func VisitorKey(r *http.Request) string {
	if c, err := r.Cookie("tcid"); err == nil && c.Value != "" {
		return c.Value
	}
	return ""
}
