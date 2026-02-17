package main

import (
	"log"
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
		log.Fatal(err)
	}

	if err := cfg.InitDataBase(); err != nil {
		log.Fatalf("데이터 베이스 연결에 실패했습니다: %v", err)
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
		BodyLimit:    5 * 1024 * 1024, // 5MB
	})
	router.Setup(app, cfg)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := app.Listen(":" + cfg.Port); err != nil {
			log.Fatalf("서버 시작에 실패했습니다: %v", err)
		}
	}()

	log.Printf("서버가 :%s 포트에서 시작되었습니다", cfg.Port)

	<-quit
	log.Println("서버를 종료합니다...")

	if err := app.Shutdown(); err != nil {
		log.Fatalf("서버 종료 실패: %v", err)
	}

	log.Println("서버가 정상 종료되었습니다")
}
