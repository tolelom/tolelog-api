package middleware

import (
	"strings"
	"tolelom_api/internal/config"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/utils"

	"github.com/gofiber/fiber/v2"
)

// OptionalAuthMiddleware 토큰이 있으면 파싱하여 userID를 context에 저장하고, 없으면 그냥 통과
func OptionalAuthMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next()
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Next()
		}

		claims, err := utils.ValidateJWT(parts[1], cfg.JWTSecret)
		if err != nil {
			return c.Next()
		}

		c.Locals("userID", claims.UserID)
		c.Locals("username", claims.Username)

		return c.Next()
	}
}

func AuthMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Error:   "missing_authorization",
				Message: "Authorization 헤더가 없습니다",
			})
		}

		// "Bearer <token>" 형식 파싱
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Error:   "invalid_token_format",
				Message: "올바른 형식은 'Bearer {token}' 입니다",
			})
		}

		tokenString := parts[1]

		// JWT 검증
		claims, err := utils.ValidateJWT(tokenString, cfg.JWTSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Error:   "invalid_token",
				Message: "유효하지 않은 토큰입니다",
			})
		}

		// Context에 사용자 정보 저장
		c.Locals("userID", claims.UserID)
		c.Locals("username", claims.Username)

		return c.Next()
	}
}
