package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/app/handler"
	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/app/service"
	pkgconfig "github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/pkg/config"
	pkgerr "github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/pkg/err"
	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/pkg/logger"
	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/pkg/middleware"
)

func main() {
	cfg, err := pkgconfig.Load()
	if err != nil {
		panic(err)
	}

	log := logger.Init(cfg.Env)
	defer func() { _ = log.Sync() }()

	// 服务与处理器
	hs := service.NewHealthService()
	healthHandler := handler.NewHealthHandler(hs)

	mux := http.NewServeMux()
	mux.Handle("/api/v1/healthz", healthHandler)

	// 构建中间件链：请求 ID -> 恢复 -> CORS -> mux
	handlerWithMiddleware := middleware.RequestID(middleware.Recovery(log)(middleware.CORS(mux)))

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: handlerWithMiddleware,
	}

	// 启动服务器
	go func() {
		log.Info("starting server", "addr", cfg.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed", "error", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("shutdown error", "error", err)
	}
	log.Info("server stopped")
	// 下面这行目前没啥用，后续想在某个 HTTP 请求context中返回标准响应
	// 或者编译时类型断言以确保函数签名符合预期啥的可以再改，不用的话连同14行一起删了
	_ = pkgerr.JSON
}
