package handler

import (
	"tolelom_api/internal/model"
	"tolelom_api/internal/service"

	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	authService *service.AuthService
}

func NewUserHandler() *UserHandler {
	return &UserHandler{
		authService: service.NewAuthService(),
	}
}

// 회원가입
func (h *UserHandler) Register(c *fiber.Ctx) error {
	var req model.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "error",
			"error":  "요청 형식이 올바르지 않습니다",
		})
	}

	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "error",
			"error":  "사용자명과 비밀번호는 필수입니다",
		})
	}

	// 비밀번호 길이 검사 (8자 이상)
	if len(req.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "error",
			"error":  "비밀번호는 8자 이상이어야 합니다",
		})
	}

	user, token, err := h.authService.RegisterUser(&req)
	if err != nil {
		if err.Error() == "user already exists" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"status": "error",
				"error":  "이미 존재하는 사용자명입니다",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "error",
			"error":  "사용자 생성에 실패했습니다",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"token":    token,
			"username": user.Username,
			"user_id":  user.ID,
		},
	})
}

// 로그인
func (h *UserHandler) Login(c *fiber.Ctx) error {
	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "error",
			"error":  "요청 형식이 올바르지 않습니다",
		})
	}

	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "error",
			"error":  "사용자명과 비밀번호는 필수입니다",
		})
	}

	user, token, err := h.authService.AuthenticateUser(&req)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status": "error",
			"error":  "사용자명 또는 비밀번호가 잘못되었습니다",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"token":    token,
			"username": user.Username,
			"user_id":  user.ID,
		},
	})
}
