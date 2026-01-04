package middleware

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"
)

func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := err.Error()

	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
		message = e.Message
	}

	logError(c, code, message, err)

	if code >= 500 {
		message = "서버 내부 오류가 발생했습니다"
	}

	return c.Status(code).JSON(fiber.Map{
		"status":  "error",
		"message": message,
	})
}

func logError(c *fiber.Ctx, code int, message string, err error) {
	log.Printf(
		"[ERROR] %d | %s | %s %s | IP: %s",
		code,
		message,
		c.Method(),
		c.Path(),
		c.IP(),
	)
}
