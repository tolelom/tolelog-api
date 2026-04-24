package middleware

import (
	"fmt"
	"log/slog"
	"runtime/debug"
	"tolelom_api/internal/dto"

	"github.com/gofiber/fiber/v2"
)

// PanicRecovery catches panics from downstream handlers, logs a structured
// error with stack trace + request context, and responds with a 500 JSON body.
// Should be registered as the first middleware so it wraps everything else.
func PanicRecovery() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				slog.Error("panic recovered",
					"panic", fmt.Sprintf("%v", r),
					"method", c.Method(),
					"path", c.Path(),
					"ip", c.IP(),
					"stack", stack,
				)
				err = c.Status(fiber.StatusInternalServerError).
					JSON(dto.NewErrorResponse("internal_error", "서버 내부 오류가 발생했습니다"))
			}
		}()
		return c.Next()
	}
}
