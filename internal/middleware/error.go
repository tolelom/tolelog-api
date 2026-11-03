package middleware

import (
	"errors"
	"log/slog"
	"strconv"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/utils"

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

	// 4xx 는 클라이언트 문제라 리포팅하면 잡음이다.
	if code >= 500 {
		utils.CaptureException(err, map[string]string{
			"method": c.Method(),
			"path":   c.Path(),
			"status": strconv.Itoa(code),
		})
		message = "서버 내부 오류가 발생했습니다"
	}

	return c.Status(code).JSON(dto.NewErrorResponse(errCode, message))
}
