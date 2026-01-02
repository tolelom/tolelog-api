package main

import (
	"log"
	"tolelom_api/internal/config"
	"tolelom_api/internal/router"

	"github.com/gofiber/fiber/v2"

	_ "tolelom_api/docs"
)

func main() {
	cfg := config.LoadConfig()

	if err := cfg.InitDataBase(); err != nil {
		log.Fatalf("데이터 베이스 연결에 실패했습니다: %v", err)
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"status":  "error",
				"message": err.Error(),
				"data":    nil,
			})
		},
	})
	router.Setup(app)

	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("서버 시작에 실패했습니다: %v", err)
	}
}
