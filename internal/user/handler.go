package user

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/validate"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	authService AuthService
	uploadDir   string
}

func NewHandler(authService AuthService, uploadDir string) *Handler {
	return &Handler{authService: authService, uploadDir: uploadDir}
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
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_request", "요청 형식이 올바르지 않습니다"))
	}
	if err := validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("validation_failed", err.Error()))
	}

	authResp, err := h.authService.RegisterUser(&req)
	if err != nil {
		if errors.Is(err, ErrUserAlreadyExists) {
			return c.Status(fiber.StatusConflict).JSON(dto.NewErrorResponse("user_already_exists", "이미 존재하는 사용자명입니다"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("registration_failed", "사용자 생성에 실패했습니다"))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResponse{
		Status: "success",
		Data: dto.AuthDataResponse{
			AccessToken:  authResp.AccessToken,
			RefreshToken: authResp.RefreshToken,
			Username:     authResp.User.Username,
			UserID:       authResp.User.ID,
			AvatarURL:    authResp.User.AvatarURL,
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
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_request", "요청 형식이 올바르지 않습니다"))
	}
	if err := validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("validation_failed", err.Error()))
	}

	authResp, err := h.authService.AuthenticateUser(&req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.NewErrorResponse("invalid_credentials", "사용자명 또는 비밀번호가 잘못되었습니다"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("login_failed", "로그인에 실패했습니다"))
	}

	return c.Status(fiber.StatusOK).JSON(dto.SuccessResponse{
		Status: "success",
		Data: dto.AuthDataResponse{
			AccessToken:  authResp.AccessToken,
			RefreshToken: authResp.RefreshToken,
			Username:     authResp.User.Username,
			UserID:       authResp.User.ID,
			AvatarURL:    authResp.User.AvatarURL,
		},
	})
}

// RefreshToken godoc
// @Summary      토큰 갱신
// @Description  리프레시 토큰으로 새로운 액세스 토큰과 리프레시 토큰을 발급합니다
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.RefreshTokenRequest  true  "리프레시 토큰"
// @Success      200   {object}  dto.SuccessResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      401   {object}  dto.ErrorResponse
// @Failure      500   {object}  dto.ErrorResponse
// @Router       /auth/refresh [post]
func (h *Handler) RefreshToken(c *fiber.Ctx) error {
	var req dto.RefreshTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_request", "요청 형식이 올바르지 않습니다"))
	}
	if err := validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("validation_failed", err.Error()))
	}

	authResp, err := h.authService.RefreshTokens(req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.NewErrorResponse("invalid_refresh_token", "유효하지 않은 리프레시 토큰입니다"))
	}

	return c.Status(fiber.StatusOK).JSON(dto.SuccessResponse{
		Status: "success",
		Data: dto.AuthDataResponse{
			AccessToken:  authResp.AccessToken,
			RefreshToken: authResp.RefreshToken,
			Username:     authResp.User.Username,
			UserID:       authResp.User.ID,
			AvatarURL:    authResp.User.AvatarURL,
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
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_user_id", "유효하지 않은 사용자 ID입니다"))
	}

	u, err := h.authService.GetUserByID(uint(userID))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.NewErrorResponse("user_not_found", "사용자를 찾을 수 없습니다"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("internal_error", "사용자 조회에 실패했습니다"))
	}

	return c.Status(fiber.StatusOK).JSON(dto.SuccessResponse{
		Status: "success",
		Data:   dto.UserToResponse(u),
	})
}

var avatarAllowedMIME = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

const avatarMaxSize = 5 * 1024 * 1024 // 5MB

func avatarMimeToExt(mime string) string {
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}

// UploadAvatar godoc
// @Summary      프로필 이미지 업로드
// @Description  프로필 이미지를 업로드합니다 (최대 5MB, jpeg/png/gif/webp)
// @Tags         Users
// @Accept       multipart/form-data
// @Produce      json
// @Param        avatar  formData  file  true  "프로필 이미지 파일"
// @Success      200     {object}  dto.SuccessResponse
// @Failure      400     {object}  dto.ErrorResponse
// @Failure      401     {object}  dto.ErrorResponse
// @Failure      500     {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /users/avatar [put]
func (h *Handler) UploadAvatar(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.NewErrorResponse("unauthorized", "인증이 필요합니다"))
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("no_file", "이미지 파일이 필요합니다"))
	}

	if file.Size > avatarMaxSize {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("file_too_large", "파일 크기는 5MB 이하여야 합니다"))
	}

	contentType := file.Header.Get("Content-Type")
	if !avatarAllowedMIME[contentType] {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_file_type", "허용되는 파일 형식: jpeg, png, gif, webp"))
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		ext = avatarMimeToExt(contentType)
	}
	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	savePath := filepath.Join(h.uploadDir, filename)
	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("upload_failed", "파일 저장에 실패했습니다"))
	}

	avatarURL := "/uploads/images/" + filename
	if err := h.authService.UpdateAvatar(userID, avatarURL); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("update_failed", "프로필 이미지 업데이트에 실패했습니다"))
	}

	return c.Status(fiber.StatusOK).JSON(dto.SuccessResponse{
		Status: "success",
		Data: fiber.Map{
			"avatar_url": avatarURL,
		},
	})
}
