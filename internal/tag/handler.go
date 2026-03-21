package tag

import (
	"tolelom_api/internal/dto"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// GetTags godoc
// @Summary      태그 목록 조회
// @Description  공개 글에 사용된 태그 목록을 사용 횟수와 함께 반환합니다
// @Tags         Tags
// @Produce      json
// @Param        sort   query  string  false  "정렬 기준: popular(기본) 또는 name"
// @Param        limit  query  int     false  "최대 반환 개수 (기본 50, 최대 200)"
// @Success      200  {object}  dto.SuccessResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /tags [get]
func (h *Handler) GetTags(c *fiber.Ctx) error {
	sort := c.Query("sort", "popular")
	if sort != "popular" && sort != "name" {
		sort = "popular"
	}

	limit := c.QueryInt("limit", 50)
	if limit < 1 || limit > 200 {
		limit = 50
	}

	result, err := h.service.GetTags(sort, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("fetch_failed", "태그 목록 조회에 실패했습니다"))
	}

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data:   result,
	})
}
