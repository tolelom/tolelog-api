package handler

import (
	"errors"
	"tolelom_api/internal/config"
	"tolelom_api/internal/model"
	"tolelom_api/internal/service"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

var userValidate = validator.New()

type UserHandler struct {
	authService *service.AuthService
}

func NewUserHandler(cfg *config.Config) *UserHandler {
	return &UserHandler{
		authService: service.NewAuthService(cfg.DB, cfg.JWTSecret),
	}
}

// Register godoc
// @Summary      회원가입
// @Description  새로운 사용자를 등록합니다
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      model.RegisterRequest  true  "회원가입 정보"
// @Success      201   {object}  model.AuthResponse
// @Failure      400   {object}  model.ErrorResponse
// @Failure      409   {object}  model.ErrorResponse
// @Failure      500   {object}  model.ErrorResponse
// @Router       /auth/register [post]
func (h *UserHandler) Register(c *fiber.Ctx) error {
	var req model.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResponse{
			Error:   "invalid_request",
			Message: "요청 형식이 올바르지 않습니다",
		})
	}
	if err := userValidate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResponse{
			Error:   "validation_failed",
			Message: err.Error(),
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

// Login godoc
// @Summary      로그인
// @Description  사용자명과 비밀번호로 로그인합니다
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      model.LoginRequest  true  "로그인 정보"
// @Success      200   {object}  model.AuthResponse
// @Failure      400   {object}  model.ErrorResponse
// @Failure      401   {object}  model.ErrorResponse
// @Failure      500   {object}  model.ErrorResponse
// @Router       /auth/login [post]
func (h *UserHandler) Login(c *fiber.Ctx) error {
	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResponse{
			Error:   "invalid_request",
			Message: "요청 형식이 올바르지 않습니다",
		})
	}
	if err := userValidate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResponse{
			Error:   "validation_failed",
			Message: err.Error(),
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
