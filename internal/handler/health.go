package handler

import "github.com/gofiber/fiber/v2"

type HealthResponse struct {
	Status  string `json:"status" example:"ok"`
	Message string `json:"message,omitempty" example:"Server is running"`
}

// HealthHandler godoc
//
//	@Summary		Health Check
//	@Description	서버의 health 상태를 반환합니다
//	@Tags			Health
//	@Produce		json
//	@Success		200	{object}	HealthResponse
//	@Router			/health [get]
func HealthHandler(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(HealthResponse{
		Status:  "ok",
		Message: "Server is running",
	})
}
