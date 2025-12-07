package handler

import (
	"strconv"
	"tolelom_api/internal/model"
	"tolelom_api/internal/service"

	"github.com/gofiber/fiber/v2"
)

type PostHandler struct {
	service *service.PostService
}

func NewPostHandler() *PostHandler {
	return &PostHandler{
		service: service.NewPostService(),
	}
}

// CreatePost - 새 글 생성
// @Summary 새 글 생성
// @Description 사용자가 새로운 글을 작성합니다
// @Tags Posts
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param body body model.CreatePostRequest true "글 내용"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /posts [post]
func (ph *PostHandler) CreatePost(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "인증 정보가 없습니다",
		})
	}

	var req model.CreatePostRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "요청 형식이 잠못되었습니다",
		})
	}

	post := &model.Post{
		Title:    req.Title,
		Content:  req.Content,
		UserID:   userID,
		IsPublic: req.IsPublic,
	}

	if err := ph.service.CreatePost(post); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "글 저장에 실패했습니다",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"id": post.ID,
		},
	})
}

// GetPost - 글 상세 조회
// @Summary 글 상세 조회
// @Description ID로 글을 조회합니다
// @Tags Posts
// @Produce json
// @Param id path int true "글 ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /posts/{id} [get]
func (ph *PostHandler) GetPost(c *fiber.Ctx) error {
	postID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "잠못된 ID입니다",
		})
	}

	post, err := ph.service.GetPostByID(uint(postID))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// 비공개 글인 경우, 작성자만 볼 수 있음
	if !post.IsPublic {
		userID, ok := c.Locals("userID").(uint)
		if !ok || userID != post.UserID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "이 글을 볼 수 없습니다",
			})
		}
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   post.ToResponse(),
	})
}

// GetPublicPosts - 공개 글 목록
// @Summary 공개 글 목록 조회
// @Description 공개된 모든 글을 페이지네이션으로 조회합니다
// @Tags Posts
// @Produce json
// @Param page query int false "페이지 번호 (기본값: 1)"
// @Param page_size query int false "페이지 크기 (기본값: 10)"
// @Success 200 {object} map[string]interface{}
// @Router /posts [get]
func (ph *PostHandler) GetPublicPosts(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("page_size", 10)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	posts, total, err := ph.service.GetPublicPosts(page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "글 목록 조회에 실패했습니다",
		})
	}

	var postResponses []model.PostListResponse
	for _, post := range posts {
		postResponses = append(postResponses, post.ToListResponse())
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   postResponses,
		"pagination": fiber.Map{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetUserPosts - 특정 사용자의 글 목록
// @Summary 사용자 글 목록 조회
// @Description 특정 사용자의 모든 글을 조회합니다
// @Tags Posts
// @Produce json
// @Param user_id path int true "사용자 ID"
// @Param page query int false "페이지 번호 (기본값: 1)"
// @Param page_size query int false "페이지 크기 (기본값: 10)"
// @Success 200 {object} map[string]interface{}
// @Router /users/{user_id}/posts [get]
func (ph *PostHandler) GetUserPosts(c *fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("user_id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "잠못된 사용자 ID입니다",
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

	posts, total, err := ph.service.GetUserPosts(uint(userID), page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "글 목록 조회에 실패했습니다",
		})
	}

	var postResponses []model.PostListResponse
	for _, post := range posts {
		postResponses = append(postResponses, post.ToListResponse())
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   postResponses,
		"pagination": fiber.Map{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// UpdatePost - 글 수정
// @Summary 글 수정
// @Description 작성자가 글을 수정합니다
// @Tags Posts
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path int true "글 ID"
// @Param body body model.UpdatePostRequest true "수정할 내용"
// @Success 200 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /posts/{id} [put]
func (ph *PostHandler) UpdatePost(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "인증 정보가 없습니다",
		})
	}

	postID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "잠못된 ID입니다",
		})
	}

	// 소유권 확인
	isOwner, err := ph.service.CheckOwnership(uint(postID), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if !isOwner {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "이 글을 수정할 권한이 없습니다",
		})
	}

	var req model.UpdatePostRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "요청 형식이 잠못되었습니다",
		})
	}

	post := &model.Post{
		Title:    req.Title,
		Content:  req.Content,
		IsPublic: req.IsPublic,
	}
	post.ID = uint(postID)

	if err := ph.service.UpdatePost(post); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "글 수정에 실패했습니다",
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "글이 수정되었습니다",
	})
}

// DeletePost - 글 삭제
// @Summary 글 삭제
// @Description 작성자가 글을 삭제합니다
// @Tags Posts
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path int true "글 ID"
// @Success 200 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /posts/{id} [delete]
func (ph *PostHandler) DeletePost(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "인증 정보가 없습니다",
		})
	}

	postID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "잠못된 ID입니다",
		})
	}

	if err := ph.service.DeletePost(uint(postID), userID); err != nil {
		if err.Error() == "글을 찾을 수 없습니다" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if err.Error() == "이 글을 삭제할 권한이 없습니다" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "글 삭제에 실패했습니다",
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "글이 삭제되었습니다",
	})
}
