package post

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

// CreatePost godoc
// @Summary      새 글 생성
// @Description  사용자가 새로운 글을 작성합니다
// @Tags         Posts
// @Accept       json
// @Produce      json
// @Param        Authorization  header  string                  true  "Bearer token"
// @Param        body           body    dto.CreatePostRequest   true  "글 내용"
// @Success      201            {object}  dto.PostResponse
// @Failure      400            {object}  dto.ErrorResponse
// @Failure      401            {object}  dto.ErrorResponse
// @Failure      500            {object}  dto.ErrorResponse
// @Router       /posts [post]
func (h *Handler) CreatePost(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
			Error:   "unauthorized",
			Message: "인증 정보가 없습니다",
		})
	}

	var req dto.CreatePostRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "요청 형식이 잘못되었습니다",
		})
	}
	if err := validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "validation_failed",
			Message: err.Error(),
		})
	}

	p := &model.Post{
		Title:    req.Title,
		Content:  req.Content,
		UserID:   userID,
		IsPublic: req.IsPublic,
	}

	if err := h.service.CreatePost(p); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "creation_failed",
			Message: "글 저장에 실패했습니다",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(dto.PostToResponse(p))
}

// GetPost godoc
// @Summary      글 상세 조회
// @Description  ID로 글을 조회합니다. 공개 글은 누구나, 비공개 글은 작성자만 조회 가능합니다.
// @Tags         Posts
// @Produce      json
// @Param        id             path    int     true   "글 ID"
// @Param        Authorization  header  string  false  "Bearer token (비공개 글 조회 시 필요)"
// @Success      200  {object}  dto.PostResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      403  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /posts/{id} [get]
func (h *Handler) GetPost(c *fiber.Ctx) error {
	postID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "잘못된 ID입니다",
		})
	}

	var currentUserID *uint
	if uid, ok := c.Locals("userID").(uint); ok {
		currentUserID = &uid
	}

	p, err := h.service.GetPostByID(uint(postID), currentUserID)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "post_not_found",
				Message: "글을 찾을 수 없습니다",
			})
		}
		if errors.Is(err, ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "이 글을 볼 권한이 없습니다",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "fetch_failed",
			Message: "글 조회에 실패했습니다",
		})
	}

	return c.JSON(dto.PostToResponse(p))
}

// GetPublicPosts godoc
// @Summary      공개 글 목록 조회
// @Description  공개된 모든 글을 페이지네이션으로 조회합니다
// @Tags         Posts
// @Produce      json
// @Param        page       query  int  false  "페이지 번호 (기본값: 1)"
// @Param        page_size  query  int  false  "페이지 크기 (기본값: 10)"
// @Success      200  {object}  dto.PostListWithPagination
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /posts [get]
func (h *Handler) GetPublicPosts(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("page_size", 10)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	posts, total, err := h.service.GetPublicPosts(page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "fetch_failed",
			Message: "글 목록 조회에 실패했습니다",
		})
	}

	var postResponses []dto.PostListResponse
	for _, p := range posts {
		postResponses = append(postResponses, dto.PostToListResponse(&p))
	}

	return c.JSON(dto.PostListWithPagination{
		Posts: postResponses,
		Pagination: dto.Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetUserPosts godoc
// @Summary      사용자 글 목록 조회
// @Description  특정 사용자의 모든 글을 조회합니다
// @Tags         Posts
// @Produce      json
// @Param        user_id    path   int  true   "사용자 ID"
// @Param        page       query  int  false  "페이지 번호 (기본값: 1)"
// @Param        page_size  query  int  false  "페이지 크기 (기본값: 10)"
// @Success      200  {object}  dto.PostListWithPagination
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /users/{user_id}/posts [get]
func (h *Handler) GetUserPosts(c *fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("user_id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "invalid_user_id",
			Message: "잘못된 사용자 ID입니다",
		})
	}

	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("page_size", 10)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	var currentUserID *uint
	if uid, ok := c.Locals("userID").(uint); ok {
		currentUserID = &uid
	}

	posts, total, err := h.service.GetUserPosts(uint(userID), currentUserID, page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "fetch_failed",
			Message: "글 목록 조회에 실패했습니다",
		})
	}

	var postResponses []dto.PostListResponse
	for _, p := range posts {
		postResponses = append(postResponses, dto.PostToListResponse(&p))
	}

	return c.JSON(dto.PostListWithPagination{
		Posts: postResponses,
		Pagination: dto.Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// UpdatePost godoc
// @Summary      글 수정
// @Description  작성자가 글을 수정합니다
// @Tags         Posts
// @Accept       json
// @Produce      json
// @Param        Authorization  header  string                  true  "Bearer token"
// @Param        id             path    int                     true  "글 ID"
// @Param        body           body    dto.UpdatePostRequest   true  "수정할 내용"
// @Success      200  {object}  dto.PostResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      403  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /posts/{id} [put]
func (h *Handler) UpdatePost(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
			Error:   "unauthorized",
			Message: "인증 정보가 없습니다",
		})
	}

	postID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "잘못된 ID입니다",
		})
	}

	var req dto.UpdatePostRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "요청 형식이 잘못되었습니다",
		})
	}
	if err := validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "validation_failed",
			Message: err.Error(),
		})
	}

	updatedPost, err := h.service.UpdatePost(uint(postID), userID, &req)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "post_not_found",
				Message: "글을 찾을 수 없습니다",
			})
		}
		if errors.Is(err, ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "이 글을 수정할 권한이 없습니다",
			})
		}
		if errors.Is(err, ErrNoFieldsToUpdate) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
				Error:   "no_fields_to_update",
				Message: "수정할 필드가 없습니다",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "update_failed",
			Message: "글 수정에 실패했습니다",
		})
	}

	return c.JSON(dto.PostToResponse(updatedPost))
}

// DeletePost godoc
// @Summary      글 삭제
// @Description  작성자가 글을 삭제합니다
// @Tags         Posts
// @Produce      json
// @Param        Authorization  header  string  true  "Bearer token"
// @Param        id             path    int     true  "글 ID"
// @Success      204  "No Content"
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      403  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /posts/{id} [delete]
func (h *Handler) DeletePost(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
			Error:   "unauthorized",
			Message: "인증 정보가 없습니다",
		})
	}

	postID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "잘못된 ID입니다",
		})
	}

	if err := h.service.DeletePost(uint(postID), userID); err != nil {
		if errors.Is(err, ErrPostNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
				Error:   "post_not_found",
				Message: "글을 찾을 수 없습니다",
			})
		}
		if errors.Is(err, ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Error:   "forbidden",
				Message: "이 글을 삭제할 권한이 없습니다",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error:   "deletion_failed",
			Message: "글 삭제에 실패했습니다",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
