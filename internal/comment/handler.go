package comment

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

// CreateComment godoc
// @Summary      댓글 작성
// @Description  게시글에 댓글을 작성합니다
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
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.NewErrorResponse("unauthorized", "인증 정보가 없습니다"))
	}

	postID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_id", "잘못된 게시글 ID입니다"))
	}

	var req dto.CreateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_request", "요청 형식이 잘못되었습니다"))
	}
	if err := validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("validation_failed", err.Error()))
	}

	comment := &model.Comment{
		PostID:  uint(postID),
		UserID:  userID,
		Content: req.Content,
	}

	if err := h.service.CreateComment(comment); err != nil {
		if errors.Is(err, ErrPostNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.NewErrorResponse("post_not_found", "게시글을 찾을 수 없습니다"))
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
// @Description  게시글의 댓글 목록을 조회합니다
// @Tags         Comments
// @Produce      json
// @Param        id  path  int  true  "게시글 ID"
// @Success      200  {object}  dto.SuccessResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /posts/{id}/comments [get]
func (h *Handler) GetComments(c *fiber.Ctx) error {
	postID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.NewErrorResponse("invalid_id", "잘못된 게시글 ID입니다"))
	}

	comments, total, err := h.service.GetCommentsByPostID(uint(postID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.NewErrorResponse("fetch_failed", "댓글 조회에 실패했습니다"))
	}

	var commentResponses []dto.CommentResponse
	for _, cm := range comments {
		commentResponses = append(commentResponses, dto.CommentToResponse(&cm))
	}

	// Ensure empty array instead of null
	if commentResponses == nil {
		commentResponses = []dto.CommentResponse{}
	}

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data: dto.CommentListResponse{
			Comments: commentResponses,
			Total:    total,
		},
	})
}

// DeleteComment godoc
// @Summary      댓글 삭제
// @Description  작성자가 댓글을 삭제합니다
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
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.NewErrorResponse("unauthorized", "인증 정보가 없습니다"))
	}

	_, err := strconv.ParseUint(c.Params("id"), 10, 32)
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

	return c.JSON(dto.SuccessResponse{
		Status: "success",
		Data: fiber.Map{
			"message": "삭제되었습니다",
		},
	})
}
