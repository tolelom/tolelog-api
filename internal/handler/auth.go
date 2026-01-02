package handler

import (
	"tolelom_api/internal/config"
	"tolelom_api/internal/model"
	"tolelom_api/internal/service"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	validate    *validator.Validate
	authService *service.AuthService
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		validate:    validator.New(),
		authService: service.NewAuthService(cfg.DB),
	}
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req model.RegisterRequest

	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if err := h.validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Validation failed: "+err.Error())
	}

	user, token, err := h.authService.RegisterUser(&req)
	if err != nil {
		switch err.Error() {
		case "user already exists":
			return fiber.NewError(fiber.StatusConflict, err.Error())
		default:
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}

	response := model.AuthResponse{
		Status: "success",
		Data: model.AuthData{
			User:  user.ToResponse(),
			Token: token,
		},
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req model.LoginRequest

	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	if err := h.validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Validation failed: "+err.Error())
	}

	user, token, err := h.authService.AuthenticateUser(&req)
	if err != nil {
		if err.Error() == "invalid credentials" {
			return fiber.NewError(fiber.StatusUnauthorized, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	response := model.AuthResponse{
		Status: "success",
		Data: model.AuthData{
			User:  user.ToResponse(),
			Token: token,
		},
	}

	return c.Status(fiber.StatusOK).JSON(response)
}
