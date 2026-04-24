// @title           Tolelog API
// @version         1.0
// @description     Tolelog 블로그 플랫폼의 REST API 서버
// @host            tolelom.xyz
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Bearer 토큰을 입력하세요 (예: Bearer eyJhbGciOi...)
package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"tolelom_api/internal/config"
	"tolelom_api/internal/middleware"
	"tolelom_api/internal/router"

	"github.com/gofiber/fiber/v2"

	_ "tolelom_api/docs"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("설정 로드 실패", "error", err)
		os.Exit(1)
	}

	if err := cfg.InitDataBase(); err != nil {
		slog.Error("데이터베이스 연결 실패", "error", err)
		os.Exit(1)
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
		BodyLimit:    5 * 1024 * 1024, // 5MB
	})
	cleanup := router.Setup(app, cfg)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := app.Listen(":" + cfg.Port); err != nil {
			slog.Error("서버 시작 실패", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("서버 시작", "port", cfg.Port)

	<-quit
	slog.Info("서버 종료 중...")

	shutdownErr := app.Shutdown()
	if shutdownErr != nil {
		slog.Error("서버 종료 실패", "error", shutdownErr)
	}

	// HTTP 서버 종료 후 리소스 정리 (Redis → DB 순)
	cleanup()
	if err := cfg.CloseDatabase(); err != nil {
		slog.Warn("DB 연결 종료 실패", "error", err)
	} else {
		slog.Info("DB 연결 종료")
	}

	if shutdownErr != nil {
		os.Exit(1)
	}
	slog.Info("서버 정상 종료")
}
