package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	focus "github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/app/focus"
	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/app/handler"
	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/app/service"
	pkgconfig "github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/pkg/config"
	pkgerr "github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/pkg/err"
	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/pkg/logger"
	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/pkg/middleware"
)

const visitorCookieName = "tcid"

// 简单工具
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func getOrSetVisitorID(w http.ResponseWriter, r *http.Request) string {
	if c, err := r.Cookie(visitorCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	vid := uuid.NewString()
	http.SetCookie(w, &http.Cookie{
		Name:     visitorCookieName,
		Value:    vid,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60 * 60 * 24 * 365, // 1年
		// Secure: true, // 上线到 https 后再打开
	})
	return vid
}

func issueGuestToken(visitorID string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-guest-secret" // 开发占位，生产换强随机
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

// /guest-login 处理器
func guestLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code": 405, "message": "请求方法有误",
		})
		return
	}

	vid := getOrSetVisitorID(w, r)
	token, err := issueGuestToken(vid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code": 500, "message": "token 错误",
		})
		return
	}
	short := vid
	if i := strings.IndexByte(vid, '-'); i > 0 {
		short = vid[:i] // 不知道怎么取名，那我就取 uuid 前缀当展示用户名算了
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token":    token,
		"username": "guest-" + short,
	})
}

func main() {
	cfg, err := pkgconfig.Load()
	if err != nil {
		panic(err)
	}

	log := logger.Init(cfg.Env)
	defer func() { _ = log.Sync() }()

	// 服务与处理器（健康检查保留）
	hs := service.NewHealthService()
	healthHandler := handler.NewHealthHandler(hs)

	mux := http.NewServeMux()
	mux.Handle("/api/v1/healthz", healthHandler)
	mux.HandleFunc("/me", meHandler)
	// 新增：游客登录（前端调用 http://localhost:3001/guest-login）
	mux.HandleFunc("/guest-login", guestLoginHandler)

	// ---- 连接数据库（从环境变量 DB_DSN 读取）
	dsn := os.Getenv("DB_DSN")
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal("db open error", "error", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		log.Fatal("db ping error", "error", err)
	}
	defer db.Close()

	// A 线接口（番茄钟与统计）
	mux.HandleFunc("/api/v1/sessions/start", focus.StartHandler(db))
	mux.HandleFunc("/api/v1/sessions/pause", focus.PauseHandler(db))
	mux.HandleFunc("/api/v1/sessions/resume", focus.ResumeHandler(db))
	mux.HandleFunc("/api/v1/sessions/finish", focus.FinishHandler(db))
	mux.HandleFunc("/api/v1/sessions/cancel", focus.CancelHandler(db))
	mux.HandleFunc("/api/v1/stats/summary", focus.SummaryHandler(db))

	// 中间件链：请求ID -> 恢复 -> CORS -> mux
	handlerWithMiddleware := middleware.RequestID(
		middleware.Recovery(log)(
			middleware.CORS(mux),
		),
	)

	srv := &http.Server{
		Addr:    cfg.Addr, // 对应前端 .env 配成 :3001
		Handler: handlerWithMiddleware,
	}

	// 启动服务器
	go func() {
		log.Info("starting server", "addr", cfg.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed", "error", err)
		}
	}()

	// 优雅关闭 (监听Ctrl + c以停止运行)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM) // 也监听系统发的SIGTERM
	<-quit
	log.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // 给5秒收尾正在处理的请求
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("shutdown error", "error", err)
	}
	log.Info("server stopped")

	// 下面这行目前没啥用，后续想在某个 HTTP 请求context中返回标准响应
	// 或者编译时类型断言以确保函数签名符合预期啥的可以再改，不用的话连同pkgerr包一起删了
	_ = pkgerr.JSON
}

// 解析 Authorization: Bearer <token>
func parseBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

func verifyGuestToken(tok string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-guest-secret"
	}
	parsed, err := jwt.Parse(tok, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !parsed.Valid {
		return "", fmt.Errorf("invalid token")
	}
	if claims, ok := parsed.Claims.(jwt.MapClaims); ok {
		if vid, _ := claims["vid"].(string); vid != "" {
			return vid, nil
		}
	}
	return "", fmt.Errorf("missing vid")
}

// GET /me
func meHandler(w http.ResponseWriter, r *http.Request) {
	// 先尝试 Bearer token
	if tok := parseBearer(r); tok != "" {
		if vid, err := verifyGuestToken(tok); err == nil {
			short := vid
			if i := strings.IndexByte(vid, '-'); i > 0 {
				short = vid[:i]
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"username":  "guest-" + short,
				"visitorId": vid,
			})
			return
		}
	}
	// 回退：cookie 里拿 tcid（对齐游客登陆）
	if c, err := r.Cookie(visitorCookieName); err == nil && c.Value != "" {
		vid := c.Value
		short := vid
		if i := strings.IndexByte(vid, '-'); i > 0 {
			short = vid[:i]
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"username":  "guest-" + short,
			"visitorId": vid,
		})
		return
	}
	writeJSON(w, http.StatusUnauthorized, map[string]any{"code": 401, "message": "unauthorized"})
}
