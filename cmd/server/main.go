package main

import (
	"log"
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
	})
	router.Setup(app, cfg)

	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("서버 시작에 실패했습니다: %v", err)
	}
}
