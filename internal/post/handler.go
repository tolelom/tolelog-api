package post

import (
	"errors"
	"strconv"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/httputil"
	"tolelom_api/internal/model"

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
	userID, err := httputil.RequireAuth(c)
	if err != nil {
		return nil
	}

	req, err := httputil.BindAndValidate[dto.CreatePostRequest](c)
	if err != nil {
		return nil
	}

	p := &model.Post{
		Title:    req.Title,
		Content:  req.Content,
		UserID:   userID,
		IsPublic: req.IsPublic,
		Status:   req.Status,
		TagsRaw:  req.Tags,
	}

	if err := h.service.CreatePost(p); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("creation_failed", "글 저장에 실패했습니다"))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResponse{
		Status: "success",
		Data:   dto.PostToResponse(p),
	})
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
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_id", "잘못된 ID입니다"))
	}

	var currentUserID *uint
	if uid, ok := c.Locals("userID").(uint); ok {
		currentUserID = &uid
	}

	p, err := h.service.GetPostByID(uint(postID), currentUserID)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.NewErrorResponse("post_not_found", "글을 찾을 수 없습니다"))
		}
		if errors.Is(err, ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(dto.NewErrorResponse("forbidden", "이 글을 볼 권한이 없습니다"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("fetch_failed", "글 조회에 실패했습니다"))
	}

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data:   dto.PostToResponse(p),
	})
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
	page, pageSize := httputil.ParsePagination(c)
	tag := c.Query("tag")

	posts, total, err := h.service.GetPublicPosts(page, pageSize, tag)
	if err != nil {
		if errors.Is(err, ErrInvalidTag) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_tag", "유효하지 않은 태그 파라미터입니다"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("fetch_failed", "글 목록 조회에 실패했습니다"))
	}

	var postResponses []dto.PostListResponse
	for _, p := range posts {
		postResponses = append(postResponses, dto.PostToListResponse(&p))
	}

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data: dto.PostListWithPagination{
			Posts:      postResponses,
			Pagination: httputil.NewPagination(page, pageSize, total),
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
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_user_id", "잘못된 사용자 ID입니다"))
	}

	page, pageSize := httputil.ParsePagination(c)
	tag := c.Query("tag")

	var currentUserID *uint
	if uid, ok := c.Locals("userID").(uint); ok {
		currentUserID = &uid
	}

	posts, total, err := h.service.GetUserPosts(uint(userID), currentUserID, page, pageSize, tag)
	if err != nil {
		if errors.Is(err, ErrInvalidTag) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_tag", "유효하지 않은 태그 파라미터입니다"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("fetch_failed", "글 목록 조회에 실패했습니다"))
	}

	var postResponses []dto.PostListResponse
	for _, p := range posts {
		postResponses = append(postResponses, dto.PostToListResponse(&p))
	}

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data: dto.PostListWithPagination{
			Posts:      postResponses,
			Pagination: httputil.NewPagination(page, pageSize, total),
		},
	})
}

// SearchPosts godoc
// @Summary      글 검색
// @Description  제목 또는 본문에서 키워드로 공개 글을 검색합니다
// @Tags         Posts
// @Produce      json
// @Param        q          query  string  true   "검색어 (2~100자)"
// @Param        page       query  int     false  "페이지 번호 (기본값: 1)"
// @Param        page_size  query  int     false  "페이지 크기 (기본값: 10)"
// @Success      200  {object}  dto.PostListWithPagination
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /posts/search [get]
func (h *Handler) SearchPosts(c *fiber.Ctx) error {
	q := c.Query("q")
	sanitized, err := SanitizeSearchQuery(q)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_query", "검색어는 2자 이상 100자 이하여야 합니다"))
	}

	page, pageSize := httputil.ParsePagination(c)

	posts, total, err := h.service.SearchPosts(sanitized, page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("search_failed", "검색에 실패했습니다"))
	}

	var postResponses []dto.PostListResponse
	for _, p := range posts {
		postResponses = append(postResponses, dto.PostToListResponse(&p))
	}

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data: dto.PostListWithPagination{
			Posts:      postResponses,
			Pagination: httputil.NewPagination(page, pageSize, total),
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
	userID, err := httputil.RequireAuth(c)
	if err != nil {
		return nil
	}

	postID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_id", "잘못된 ID입니다"))
	}

	req, err := httputil.BindAndValidate[dto.UpdatePostRequest](c)
	if err != nil {
		return nil
	}

	updatedPost, err := h.service.UpdatePost(uint(postID), userID, req)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.NewErrorResponse("post_not_found", "글을 찾을 수 없습니다"))
		}
		if errors.Is(err, ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(dto.NewErrorResponse("forbidden", "이 글을 수정할 권한이 없습니다"))
		}
		if errors.Is(err, ErrNoFieldsToUpdate) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("no_fields_to_update", "수정할 필드가 없습니다"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("update_failed", "글 수정에 실패했습니다"))
	}

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data:   dto.PostToResponse(updatedPost),
	})
}

// DeletePost godoc
// @Summary      글 삭제
// @Description  작성자가 글을 삭제합니다
// @Tags         Posts
// @Produce      json
// @Param        Authorization  header  string  true  "Bearer token"
// @Param        id             path    int     true  "글 ID"
// @Success      204  "삭제 성공"
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      403  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /posts/{id} [delete]
func (h *Handler) DeletePost(c *fiber.Ctx) error {
	userID, err := httputil.RequireAuth(c)
	if err != nil {
		return nil
	}

	postID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_id", "잘못된 ID입니다"))
	}

	if err := h.service.DeletePost(uint(postID), userID); err != nil {
		if errors.Is(err, ErrPostNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.NewErrorResponse("post_not_found", "글을 찾을 수 없습니다"))
		}
		if errors.Is(err, ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(dto.NewErrorResponse("forbidden", "이 글을 삭제할 권한이 없습니다"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("deletion_failed", "글 삭제에 실패했습니다"))
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ToggleLike godoc
// @Summary      좋아요 토글
// @Description  게시글의 좋아요를 토글합니다. 이미 좋아요한 경우 취소, 아닌 경우 추가됩니다.
// @Tags         Posts
// @Produce      json
// @Param        Authorization  header  string  true  "Bearer token"
// @Param        id             path    int     true  "글 ID"
// @Success      200  {object}  dto.SuccessResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /posts/{id}/like [post]
func (h *Handler) ToggleLike(c *fiber.Ctx) error {
	userID, err := httputil.RequireAuth(c)
	if err != nil {
		return nil
	}

	postID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_id", "잘못된 ID입니다"))
	}

	liked, likeCount, err := h.service.ToggleLike(uint(postID), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("like_failed", "좋아요 처리에 실패했습니다"))
	}

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data: fiber.Map{
			"liked":      liked,
			"like_count": likeCount,
		},
	})
}

// GetLikeStatus godoc
// @Summary      좋아요 상태 조회
// @Description  현재 사용자가 해당 게시글을 좋아요했는지 확인합니다
// @Tags         Posts
// @Produce      json
// @Param        id             path    int     true   "글 ID"
// @Param        Authorization  header  string  false  "Bearer token (로그인 시)"
// @Success      200  {object}  dto.SuccessResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /posts/{id}/like [get]
func (h *Handler) GetLikeStatus(c *fiber.Ctx) error {
	postID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_id", "잘못된 ID입니다"))
	}

	liked := false
	if userID, ok := c.Locals("userID").(uint); ok {
		liked = h.service.IsLiked(uint(postID), userID)
	}

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data: fiber.Map{
			"liked": liked,
		},
	})
}

// GetDrafts godoc
// @Summary      내 초안 목록 조회
// @Description  현재 로그인한 사용자의 초안 글 목록을 조회합니다
// @Tags         Posts
// @Produce      json
// @Success      200  {object}  dto.SuccessResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /drafts [get]
func (h *Handler) GetDrafts(c *fiber.Ctx) error {
	userID, err := httputil.RequireAuth(c)
	if err != nil {
		return nil
	}
	posts, err := h.service.GetDrafts(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("internal_error", "초안 목록 조회에 실패했습니다"))
	}
	responses := make([]dto.PostListResponse, len(posts))
	for i := range posts {
		responses[i] = dto.PostToListResponse(&posts[i])
	}
	return c.Status(fiber.StatusOK).JSON(dto.SuccessResponse{
		Status: "success",
		Data:   responses,
	})
}
