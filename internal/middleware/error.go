package middleware

import (
	"errors"
	"log/slog"
	"tolelom_api/internal/dto"

	"github.com/gofiber/fiber/v2"
)

func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := err.Error()
	errCode := "internal_error"

	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
		message = e.Message
		errCode = "http_error"
	}

	slog.Error("request error",
		"status", code,
		"error", errCode,
		"message", message,
		"method", c.Method(),
		"path", c.Path(),
		"ip", c.IP(),
	)

	if code >= 500 {
		message = "서버 내부 오류가 발생했습니다"
	}

	return c.Status(code).JSON(dto.NewErrorResponse(errCode, message))
}
