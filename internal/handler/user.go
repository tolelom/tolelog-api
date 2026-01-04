package handler

import (
	"errors"
	"tolelom_api/internal/config"
	"tolelom_api/internal/model"
	"tolelom_api/internal/service"

	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	authService *service.AuthService
}

func NewUserHandler(cfg *config.Config) *UserHandler {
	return &UserHandler{
		authService: service.NewAuthService(cfg.DB),
	}
}

// 회원가입
func (h *UserHandler) Register(c *fiber.Ctx) error {
	var req model.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResponse{
			Error:   "invalid_request",
			Message: "요청 형식이 올바르지 않습니다",
		})
	}

	authResp, err := h.authService.RegisterUser(&req)
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			return c.Status(fiber.StatusConflict).JSON(model.ErrorResponse{
				Error:   "user_already_exists",
				Message: "이미 존재하는 사용자명입니다",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResponse{
			Error:   "registration_failed",
			Message: "사용자 생성에 실패했습니다",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(authResp)
}

// 로그인
func (h *UserHandler) Login(c *fiber.Ctx) error {
	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResponse{
			Error:   "invalid_request",
			Message: "요청 형식이 올바르지 않습니다",
		})
	}

	authResp, err := h.authService.AuthenticateUser(&req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return c.Status(fiber.StatusUnauthorized).JSON(model.ErrorResponse{
				Error:   "invalid_credentials",
				Message: "사용자명 또는 비밀번호가 잘못되었습니다",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResponse{
			Error:   "login_failed",
			Message: "로그인에 실패했습니다",
		})
	}

	return c.Status(fiber.StatusOK).JSON(authResp)
}
