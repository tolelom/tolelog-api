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

// CreateSeries godoc
// @Summary      시리즈 생성
// @Description  새로운 시리즈를 생성합니다
// @Tags         Series
// @Accept       json
// @Produce      json
// @Param        Authorization  header  string                    true  "Bearer token"
// @Param        body           body    dto.CreateSeriesRequest   true  "시리즈 정보"
// @Success      201  {object}  dto.SuccessResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /series [post]
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

// GetSeries godoc
// @Summary      시리즈 상세 조회
// @Description  시리즈 ID로 시리즈 상세 정보와 포함된 글 목록을 조회합니다
// @Tags         Series
// @Produce      json
// @Param        id  path  int  true  "시리즈 ID"
// @Success      200  {object}  dto.SuccessResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /series/{id} [get]
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

// GetUserSeries godoc
// @Summary      사용자 시리즈 목록 조회
// @Description  특정 사용자의 시리즈 목록을 조회합니다
// @Tags         Series
// @Produce      json
// @Param        user_id  path  int  true  "사용자 ID"
// @Success      200  {object}  dto.SuccessResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /users/{user_id}/series [get]
func (h *Handler) GetUserSeries(c *fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("user_id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_id", "잘못된 사용자 ID입니다"))
	}

	seriesList, postCounts, err := h.service.GetSeriesByUserID(uint(userID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("fetch_failed", "시리즈 목록 조회에 실패했습니다"))
	}

	responses := make([]dto.SeriesResponse, len(seriesList))
	for i, s := range seriesList {
		responses[i] = dto.SeriesToResponse(&s, int(postCounts[s.ID]))
	}

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data:   responses,
	})
}

// UpdateSeries godoc
// @Summary      시리즈 수정
// @Description  작성자가 시리즈 정보를 수정합니다
// @Tags         Series
// @Accept       json
// @Produce      json
// @Param        Authorization  header  string                    true  "Bearer token"
// @Param        id             path    int                       true  "시리즈 ID"
// @Param        body           body    dto.UpdateSeriesRequest   true  "수정할 시리즈 정보"
// @Success      200  {object}  dto.SuccessResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      403  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /series/{id} [put]
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

// DeleteSeries godoc
// @Summary      시리즈 삭제
// @Description  작성자가 시리즈를 삭제합니다. 포함된 글은 삭제되지 않습니다.
// @Tags         Series
// @Produce      json
// @Param        Authorization  header  string  true  "Bearer token"
// @Param        id             path    int     true  "시리즈 ID"
// @Success      200  {object}  dto.SuccessResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      403  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /series/{id} [delete]
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

// AddPostToSeries godoc
// @Summary      시리즈에 글 추가
// @Description  시리즈에 글을 추가합니다. 시리즈와 글 모두 본인 소유여야 합니다.
// @Tags         Series
// @Accept       json
// @Produce      json
// @Param        Authorization  header  string                       true  "Bearer token"
// @Param        id             path    int                          true  "시리즈 ID"
// @Param        body           body    dto.AddPostToSeriesRequest   true  "추가할 글 정보"
// @Success      200  {object}  dto.SuccessResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      403  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /series/{id}/posts [post]
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

// RemovePostFromSeries godoc
// @Summary      시리즈에서 글 제거
// @Description  시리즈에서 글을 제거합니다. 글 자체는 삭제되지 않습니다.
// @Tags         Series
// @Produce      json
// @Param        Authorization  header  string  true  "Bearer token"
// @Param        id             path    int     true  "시리즈 ID"
// @Param        post_id        path    int     true  "게시글 ID"
// @Success      200  {object}  dto.SuccessResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      403  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /series/{id}/posts/{post_id} [delete]
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

// ReorderPosts godoc
// @Summary      시리즈 글 순서 변경
// @Description  시리즈에 포함된 글의 순서를 변경합니다
// @Tags         Series
// @Accept       json
// @Produce      json
// @Param        Authorization  header  string                     true  "Bearer token"
// @Param        id             path    int                        true  "시리즈 ID"
// @Param        body           body    dto.ReorderPostsRequest    true  "새로운 글 순서"
// @Success      200  {object}  dto.SuccessResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      403  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /series/{id}/reorder [put]
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

// GetSeriesNavigation godoc
// @Summary      시리즈 네비게이션 조회
// @Description  게시글이 속한 시리즈의 이전/다음 글 정보를 조회합니다
// @Tags         Series
// @Produce      json
// @Param        id  path  int  true  "게시글 ID"
// @Success      200  {object}  dto.SuccessResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /posts/{id}/series-nav [get]
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
