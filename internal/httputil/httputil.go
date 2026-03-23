package httputil

import (
	"errors"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/validate"

	"github.com/gofiber/fiber/v2"
)

// ErrResponseSent is returned when the HTTP response has already been sent.
var ErrResponseSent = errors.New("response already sent")

// ParsePagination extracts page and pageSize from query parameters with defaults and bounds.
func ParsePagination(c *fiber.Ctx) (page, pageSize int) {
	page = c.QueryInt("page", 1)
	pageSize = c.QueryInt("page_size", 10)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return
}

// BindAndValidate parses the request body into T and validates it.
// On failure it sends a JSON error response and returns ErrResponseSent.
func BindAndValidate[T any](c *fiber.Ctx) (*T, error) {
	var req T
	if err := c.BodyParser(&req); err != nil {
		_ = c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_request", "요청 형식이 잘못되었습니다"))
		return nil, ErrResponseSent
	}
	if err := validate.Struct(&req); err != nil {
		_ = c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("validation_failed", err.Error()))
		return nil, ErrResponseSent
	}
	return &req, nil
}

// RequireAuth extracts the authenticated userID from Fiber locals.
// On failure it sends a 401 response and returns ErrResponseSent.
func RequireAuth(c *fiber.Ctx) (uint, error) {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		_ = c.Status(fiber.StatusUnauthorized).JSON(dto.NewErrorResponse("unauthorized", "인증 정보가 없습니다"))
		return 0, ErrResponseSent
	}
	return userID, nil
}

// NewPagination creates a Pagination struct from page, pageSize, and total.
func NewPagination(page, pageSize int, total int64) dto.Pagination {
	return dto.Pagination{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: (total + int64(pageSize) - 1) / int64(pageSize),
	}
}
