package comment

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

// CreateComment godoc
// @Summary      댓글 작성
// @Description  게시글에 댓글을 작성합니다. parent_id를 지정하면 대댓글이 됩니다.
// @Tags         Comments
// @Accept       json
// @Produce      json
// @Param        Authorization  header  string                      true  "Bearer token"
// @Param        id             path    int                         true  "게시글 ID"
// @Param        body           body    dto.CreateCommentRequest    true  "댓글 내용"
// @Success      201  {object}  dto.SuccessResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /posts/{id}/comments [post]
func (h *Handler) CreateComment(c *fiber.Ctx) error {
	userID, err := httputil.RequireAuth(c)
	if err != nil {
		return nil
	}

	postID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_id", "잘못된 게시글 ID입니다"))
	}

	req, err := httputil.BindAndValidate[dto.CreateCommentRequest](c)
	if err != nil {
		return nil
	}

	comment := &model.Comment{
		PostID:   uint(postID),
		UserID:   userID,
		Content:  req.Content,
		ParentID: req.ParentID,
	}

	if err := h.service.CreateComment(comment); err != nil {
		if errors.Is(err, ErrPostNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.NewErrorResponse("post_not_found", "게시글을 찾을 수 없습니다"))
		}
		if errors.Is(err, ErrParentNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.NewErrorResponse("parent_not_found", "부모 댓글을 찾을 수 없습니다"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("creation_failed", "댓글 저장에 실패했습니다"))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResponse{
		Status: "success",
		Data:   dto.CommentToResponse(comment),
	})
}

// GetComments godoc
// @Summary      댓글 목록 조회
// @Description  게시글의 댓글 목록을 트리 구조로 조회합니다
// @Tags         Comments
// @Produce      json
// @Param        id     path   int  true   "게시글 ID"
// @Param        limit  query  int  false  "최대 댓글 수 (기본값: 200, 최대: 500)"
// @Success      200  {object}  dto.SuccessResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /posts/{id}/comments [get]
func (h *Handler) GetComments(c *fiber.Ctx) error {
	postID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_id", "잘못된 게시글 ID입니다"))
	}

	limit := c.QueryInt("limit", 200)
	if limit < 1 || limit > 500 {
		limit = 200
	}

	comments, total, err := h.service.GetCommentsByPostID(uint(postID), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("fetch_failed", "댓글 조회에 실패했습니다"))
	}

	tree := dto.BuildCommentTree(comments)

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data: dto.CommentListResponse{
			Comments: tree,
			Total:    total,
		},
	})
}

// UpdateComment godoc
// @Summary      댓글 수정
// @Description  작성자가 댓글을 수정합니다
// @Tags         Comments
// @Accept       json
// @Produce      json
// @Param        Authorization  header  string                      true  "Bearer token"
// @Param        id             path    int                         true  "게시글 ID"
// @Param        comment_id     path    int                         true  "댓글 ID"
// @Param        body           body    dto.UpdateCommentRequest    true  "수정할 댓글 내용"
// @Success      200  {object}  dto.SuccessResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      403  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /posts/{id}/comments/{comment_id} [put]
func (h *Handler) UpdateComment(c *fiber.Ctx) error {
	userID, err := httputil.RequireAuth(c)
	if err != nil {
		return nil
	}

	_, err = strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_id", "잘못된 게시글 ID입니다"))
	}

	commentID, err := strconv.ParseUint(c.Params("comment_id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_comment_id", "잘못된 댓글 ID입니다"))
	}

	req, err := httputil.BindAndValidate[dto.UpdateCommentRequest](c)
	if err != nil {
		return nil
	}

	comment, err := h.service.UpdateComment(uint(commentID), userID, req.Content)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.NewErrorResponse("comment_not_found", "댓글을 찾을 수 없습니다"))
		}
		if errors.Is(err, ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(dto.NewErrorResponse("forbidden", "이 댓글을 수정할 권한이 없습니다"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("update_failed", "댓글 수정에 실패했습니다"))
	}

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data:   dto.CommentToResponse(comment),
	})
}

// DeleteComment godoc
// @Summary      댓글 삭제
// @Description  작성자가 댓글을 삭제합니다. 하위 대댓글은 유지됩니다.
// @Tags         Comments
// @Produce      json
// @Param        Authorization  header  string  true  "Bearer token"
// @Param        id             path    int     true  "게시글 ID"
// @Param        comment_id     path    int     true  "댓글 ID"
// @Success      200  {object}  dto.SuccessResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      403  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /posts/{id}/comments/{comment_id} [delete]
func (h *Handler) DeleteComment(c *fiber.Ctx) error {
	userID, err := httputil.RequireAuth(c)
	if err != nil {
		return nil
	}

	_, err = strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_id", "잘못된 게시글 ID입니다"))
	}

	commentID, err := strconv.ParseUint(c.Params("comment_id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_comment_id", "잘못된 댓글 ID입니다"))
	}

	if err := h.service.DeleteComment(uint(commentID), userID); err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.NewErrorResponse("comment_not_found", "댓글을 찾을 수 없습니다"))
		}
		if errors.Is(err, ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(dto.NewErrorResponse("forbidden", "이 댓글을 삭제할 권한이 없습니다"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("deletion_failed", "댓글 삭제에 실패했습니다"))
	}

	return c.SendStatus(fiber.StatusNoContent)
}
