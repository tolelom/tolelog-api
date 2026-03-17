package series

import (
	"errors"
	"strconv"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/model"
	"tolelom_api/internal/validate"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateSeries(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.NewErrorResponse("unauthorized", "인증 정보가 없습니다"))
	}

	var req dto.CreateSeriesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_request", "요청 형식이 잘못되었습니다"))
	}
	if err := validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("validation_failed", err.Error()))
	}

	series := &model.Series{
		Title:       req.Title,
		Description: req.Description,
		UserID:      userID,
	}

	if err := h.service.CreateSeries(series); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("creation_failed", "시리즈 생성에 실패했습니다"))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResponse{
		Status: "success",
		Data:   dto.SeriesToResponse(series, 0),
	})
}

func (h *Handler) GetSeries(c *fiber.Ctx) error {
	seriesID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_id", "잘못된 시리즈 ID입니다"))
	}

	series, posts, err := h.service.GetSeriesByID(uint(seriesID))
	if err != nil {
		if errors.Is(err, ErrSeriesNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.NewErrorResponse("series_not_found", "시리즈를 찾을 수 없습니다"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("fetch_failed", "시리즈 조회에 실패했습니다"))
	}

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data:   dto.SeriesToDetailResponse(series, posts),
	})
}

func (h *Handler) GetUserSeries(c *fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("user_id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_id", "잘못된 사용자 ID입니다"))
	}

	seriesList, err := h.service.GetSeriesByUserID(uint(userID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("fetch_failed", "시리즈 목록 조회에 실패했습니다"))
	}

	responses := make([]dto.SeriesResponse, len(seriesList))
	for i, s := range seriesList {
		responses[i] = dto.SeriesToResponse(&s, 0)
		count, err := h.service.CountPostsInSeries(s.ID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("fetch_failed", "글 수 조회에 실패했습니다"))
		}
		responses[i].PostCount = int(count)
	}

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data:   responses,
	})
}

func (h *Handler) UpdateSeries(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.NewErrorResponse("unauthorized", "인증 정보가 없습니다"))
	}

	seriesID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_id", "잘못된 시리즈 ID입니다"))
	}

	var req dto.UpdateSeriesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_request", "요청 형식이 잘못되었습니다"))
	}
	if err := validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("validation_failed", err.Error()))
	}

	series, err := h.service.UpdateSeries(uint(seriesID), userID, &req)
	if err != nil {
		if errors.Is(err, ErrSeriesNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.NewErrorResponse("series_not_found", "시리즈를 찾을 수 없습니다"))
		}
		if errors.Is(err, ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(dto.NewErrorResponse("forbidden", "이 시리즈를 수정할 권한이 없습니다"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("update_failed", "시리즈 수정에 실패했습니다"))
	}

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data:   dto.SeriesToResponse(series, 0),
	})
}

func (h *Handler) DeleteSeries(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.NewErrorResponse("unauthorized", "인증 정보가 없습니다"))
	}

	seriesID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_id", "잘못된 시리즈 ID입니다"))
	}

	if err := h.service.DeleteSeries(uint(seriesID), userID); err != nil {
		if errors.Is(err, ErrSeriesNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.NewErrorResponse("series_not_found", "시리즈를 찾을 수 없습니다"))
		}
		if errors.Is(err, ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(dto.NewErrorResponse("forbidden", "이 시리즈를 삭제할 권한이 없습니다"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("deletion_failed", "시리즈 삭제에 실패했습니다"))
	}

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data:   fiber.Map{"message": "삭제되었습니다"},
	})
}

func (h *Handler) AddPostToSeries(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.NewErrorResponse("unauthorized", "인증 정보가 없습니다"))
	}

	seriesID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_id", "잘못된 시리즈 ID입니다"))
	}

	var req dto.AddPostToSeriesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_request", "요청 형식이 잘못되었습니다"))
	}
	if err := validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("validation_failed", err.Error()))
	}

	if err := h.service.AddPostToSeries(uint(seriesID), req.PostID, req.Order, userID); err != nil {
		if errors.Is(err, ErrSeriesNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.NewErrorResponse("series_not_found", "시리즈를 찾을 수 없습니다"))
		}
		if errors.Is(err, ErrPostNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.NewErrorResponse("post_not_found", "게시글을 찾을 수 없습니다"))
		}
		if errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrPostNotOwned) {
			return c.Status(fiber.StatusForbidden).JSON(dto.NewErrorResponse("forbidden", "권한이 없습니다"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("add_failed", "글 추가에 실패했습니다"))
	}

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data:   fiber.Map{"message": "추가되었습니다"},
	})
}

func (h *Handler) RemovePostFromSeries(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.NewErrorResponse("unauthorized", "인증 정보가 없습니다"))
	}

	seriesID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_id", "잘못된 시리즈 ID입니다"))
	}

	postID, err := strconv.ParseUint(c.Params("post_id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_post_id", "잘못된 게시글 ID입니다"))
	}

	if err := h.service.RemovePostFromSeries(uint(seriesID), uint(postID), userID); err != nil {
		if errors.Is(err, ErrSeriesNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.NewErrorResponse("series_not_found", "시리즈를 찾을 수 없습니다"))
		}
		if errors.Is(err, ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(dto.NewErrorResponse("forbidden", "권한이 없습니다"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("remove_failed", "글 제거에 실패했습니다"))
	}

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data:   fiber.Map{"message": "제거되었습니다"},
	})
}

func (h *Handler) ReorderPosts(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.NewErrorResponse("unauthorized", "인증 정보가 없습니다"))
	}

	seriesID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_id", "잘못된 시리즈 ID입니다"))
	}

	var req dto.ReorderPostsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_request", "요청 형식이 잘못되었습니다"))
	}
	if err := validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("validation_failed", err.Error()))
	}

	if err := h.service.ReorderPosts(uint(seriesID), req.PostIDs, userID); err != nil {
		if errors.Is(err, ErrSeriesNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.NewErrorResponse("series_not_found", "시리즈를 찾을 수 없습니다"))
		}
		if errors.Is(err, ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(dto.NewErrorResponse("forbidden", "권한이 없습니다"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("reorder_failed", "순서 변경에 실패했습니다"))
	}

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data:   fiber.Map{"message": "순서가 변경되었습니다"},
	})
}

func (h *Handler) GetSeriesNavigation(c *fiber.Ctx) error {
	postID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_id", "잘못된 게시글 ID입니다"))
	}

	nav, err := h.service.GetSeriesNavigation(uint(postID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("fetch_failed", "시리즈 네비게이션 조회에 실패했습니다"))
	}

	if nav == nil {
		return c.JSON(dto.SuccessResponse{
			Status: "success",
			Data:   nil,
		})
	}

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data:   nav,
	})
}
