package httputil

import (
	"errors"
	"path/filepath"
	"strconv"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/upload"
	"tolelom_api/internal/validate"

	"github.com/gofiber/fiber/v2"
)

// ErrResponseSent means the helper has already written the error response; the handler should just return nil.
var ErrResponseSent = errors.New("response already sent")

const (
	userIDKey       = "userID"
	defaultPageSize = 10
	maxPageSize     = 100
)

func SetUserID(c *fiber.Ctx, userID uint) {
	c.Locals(userIDKey, userID)
}

func ParsePagination(c *fiber.Ctx) (page, pageSize int) {
	page = c.QueryInt("page", 1)
	pageSize = c.QueryInt("page_size", defaultPageSize)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > maxPageSize {
		pageSize = defaultPageSize
	}
	return page, pageSize
}

func BindAndValidate[T any](c *fiber.Ctx) (*T, error) {
	return BindAndValidateWithMessage[T](c, "요청 형식이 잘못되었습니다")
}

func BindAndValidateWithMessage[T any](c *fiber.Ctx, malformedMessage string) (*T, error) {
	var req T
	if err := c.BodyParser(&req); err != nil {
		_ = c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_request", malformedMessage))
		return nil, ErrResponseSent
	}
	if err := validate.Struct(&req); err != nil {
		_ = c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("validation_failed", err.Error()))
		return nil, ErrResponseSent
	}
	return &req, nil
}

func RequireAuth(c *fiber.Ctx) (uint, error) {
	userID, ok := c.Locals(userIDKey).(uint)
	if !ok {
		_ = c.Status(fiber.StatusUnauthorized).JSON(dto.NewErrorResponse("unauthorized", "인증 정보가 없습니다"))
		return 0, ErrResponseSent
	}
	return userID, nil
}

func ParseID(c *fiber.Ctx, param, code, message string) (uint, error) {
	id, err := strconv.ParseUint(c.Params(param), 10, 32)
	if err != nil {
		_ = c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse(code, message))
		return 0, ErrResponseSent
	}
	return uint(id), nil
}

func OptionalUserID(c *fiber.Ctx) *uint {
	if userID, ok := c.Locals(userIDKey).(uint); ok {
		return &userID
	}
	return nil
}

func NewPagination(page, pageSize int, total int64) dto.Pagination {
	return dto.Pagination{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: (total + int64(pageSize) - 1) / int64(pageSize),
	}
}

// SaveImageUpload returns the public URL of the stored image.
func SaveImageUpload(c *fiber.Ctx, field, dir string) (string, error) {
	fileHeader, err := c.FormFile(field)
	if err != nil {
		_ = c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("no_file", "이미지 파일이 필요합니다"))
		return "", ErrResponseSent
	}

	file, err := fileHeader.Open()
	if err != nil {
		_ = c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("file_open_failed", "파일을 열 수 없습니다"))
		return "", ErrResponseSent
	}
	defer func() { _ = file.Close() }()

	filename, err := upload.ValidateAndGenerateFilename(fileHeader, file)
	if err != nil {
		_ = c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_file", err.Error()))
		return "", ErrResponseSent
	}

	if err := c.SaveFile(fileHeader, filepath.Join(dir, filename)); err != nil {
		_ = c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("upload_failed", "파일 저장에 실패했습니다"))
		return "", ErrResponseSent
	}
	return upload.PublicURL(filename), nil
}
