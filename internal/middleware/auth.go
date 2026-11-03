package middleware

import (
	"strings"
	"tolelom_api/internal/auth"
	"tolelom_api/internal/config"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/httputil"

	"github.com/gofiber/fiber/v2"
)

// OptionalAuthMiddleware treats a missing or invalid token as an anonymous request.
func OptionalAuthMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := bearerToken(c.Get(fiber.HeaderAuthorization))
		if !ok {
			return c.Next()
		}
		if claims, err := auth.ValidateAccessToken(token, cfg.JWTSecret); err == nil {
			httputil.SetUserID(c, claims.UserID)
		}
		return c.Next()
	}
}

func AuthMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get(fiber.HeaderAuthorization)
		if header == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.NewErrorResponse("missing_authorization", "Authorization 헤더가 없습니다"))
		}

		token, ok := bearerToken(header)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.NewErrorResponse("invalid_token_format", "올바른 형식은 'Bearer {token}' 입니다"))
		}

		claims, err := auth.ValidateAccessToken(token, cfg.JWTSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.NewErrorResponse("invalid_token", "유효하지 않은 토큰입니다"))
		}

		httputil.SetUserID(c, claims.UserID)
		return c.Next()
	}
}

func bearerToken(header string) (string, bool) {
	parts := strings.Split(header, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", false
	}
	return parts[1], true
}
