package user

import (
	"errors"
	"strconv"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/validate"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	authService AuthService
}

func NewHandler(authService AuthService) *Handler {
	return &Handler{authService: authService}
}

// Register godoc
// @Summary      회원가입
// @Description  새로운 사용자를 등록합니다
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.RegisterRequest  true  "회원가입 정보"
// @Success      201   {object}  dto.AuthResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      409   {object}  dto.ErrorResponse
// @Failure      500   {object}  dto.ErrorResponse
// @Router       /auth/register [post]
func (h *Handler) Register(c *fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "요청 형식이 올바르지 않습니다",
		})
	}
	if err := validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "validation_failed",
			Message: err.Error(),
		})
	}

	authResp, err := h.authService.RegisterUser(&req)
	if err != nil {
		if errors.Is(err, ErrUserAlreadyExists) {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{
				Error:   "user_already_exists",
				Message: "이미 존재하는 사용자명입니다",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "registration_failed",
			Message: "사용자 생성에 실패했습니다",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResponse{
		Status: "success",
		Data: dto.AuthDataResponse{
			Token:    authResp.Token,
			Username: authResp.User.Username,
			UserID:   authResp.User.ID,
		},
	})
}

// Login godoc
// @Summary      로그인
// @Description  사용자명과 비밀번호로 로그인합니다
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.LoginRequest  true  "로그인 정보"
// @Success      200   {object}  dto.AuthResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      401   {object}  dto.ErrorResponse
// @Failure      500   {object}  dto.ErrorResponse
// @Router       /auth/login [post]
func (h *Handler) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "요청 형식이 올바르지 않습니다",
		})
	}
	if err := validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "validation_failed",
			Message: err.Error(),
		})
	}

	authResp, err := h.authService.AuthenticateUser(&req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Error:   "invalid_credentials",
				Message: "사용자명 또는 비밀번호가 잘못되었습니다",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "login_failed",
			Message: "로그인에 실패했습니다",
		})
	}

	return c.Status(fiber.StatusOK).JSON(dto.SuccessResponse{
		Status: "success",
		Data: dto.AuthDataResponse{
			Token:    authResp.Token,
			Username: authResp.User.Username,
			UserID:   authResp.User.ID,
		},
	})
}

// GetProfile godoc
// @Summary      유저 프로필 조회
// @Description  유저 ID로 프로필 정보를 조회합니다
// @Tags         Users
// @Produce      json
// @Param        user_id  path      int  true  "유저 ID"
// @Success      200      {object}  dto.SuccessResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      404      {object}  dto.ErrorResponse
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /users/{user_id} [get]
func (h *Handler) GetProfile(c *fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("user_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "invalid_user_id",
			Message: "유효하지 않은 사용자 ID입니다",
		})
	}

	u, err := h.authService.GetUserByID(uint(userID))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "user_not_found",
				Message: "사용자를 찾을 수 없습니다",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "internal_error",
			Message: "사용자 조회에 실패했습니다",
		})
	}

	return c.Status(fiber.StatusOK).JSON(dto.SuccessResponse{
		Status: "success",
		Data:   dto.UserToResponse(u),
	})
}
