package post

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/model"

	"github.com/gofiber/fiber/v2"
)

// mockService implements Service for testing.
type mockService struct {
	createPostFn     func(post *model.Post) error
	getPostByIDFn    func(postID uint, userID *uint) (*model.Post, error)
	getPublicPostsFn func(page, pageSize int, tag string) ([]model.Post, int64, error)
	getUserPostsFn   func(userID uint, currentUserID *uint, page, pageSize int, tag string) ([]model.Post, int64, error)
	updatePostFn     func(postID uint, userID uint, req *dto.UpdatePostRequest) (*model.Post, error)
	deletePostFn     func(postID uint, userID uint) error
	searchPostsFn    func(query string, page, pageSize int) ([]model.Post, int64, error)
	toggleLikeFn     func(postID uint, userID uint) (bool, uint, error)
	isLikedFn        func(postID uint, userID uint) bool
	getDraftsFn      func(userID uint) ([]model.Post, error)
}

func (m *mockService) CreatePost(post *model.Post) error {
	if m.createPostFn != nil {
		return m.createPostFn(post)
	}
	return nil
}

func (m *mockService) GetPostByID(postID uint, userID *uint) (*model.Post, error) {
	if m.getPostByIDFn != nil {
		return m.getPostByIDFn(postID, userID)
	}
	return &model.Post{ID: postID}, nil
}

func (m *mockService) GetPublicPosts(page, pageSize int, tag string) ([]model.Post, int64, error) {
	if m.getPublicPostsFn != nil {
		return m.getPublicPostsFn(page, pageSize, tag)
	}
	return []model.Post{}, 0, nil
}

func (m *mockService) GetUserPosts(userID uint, currentUserID *uint, page, pageSize int, tag string) ([]model.Post, int64, error) {
	if m.getUserPostsFn != nil {
		return m.getUserPostsFn(userID, currentUserID, page, pageSize, tag)
	}
	return []model.Post{}, 0, nil
}

func (m *mockService) UpdatePost(postID uint, userID uint, req *dto.UpdatePostRequest) (*model.Post, error) {
	if m.updatePostFn != nil {
		return m.updatePostFn(postID, userID, req)
	}
	return &model.Post{ID: postID}, nil
}

func (m *mockService) DeletePost(postID uint, userID uint) error {
	if m.deletePostFn != nil {
		return m.deletePostFn(postID, userID)
	}
	return nil
}

func (m *mockService) SearchPosts(query string, page, pageSize int) ([]model.Post, int64, error) {
	if m.searchPostsFn != nil {
		return m.searchPostsFn(query, page, pageSize)
	}
	return []model.Post{}, 0, nil
}

func (m *mockService) ToggleLike(postID uint, userID uint) (bool, uint, error) {
	if m.toggleLikeFn != nil {
		return m.toggleLikeFn(postID, userID)
	}
	return false, 0, nil
}

func (m *mockService) IsLiked(postID uint, userID uint) bool {
	if m.isLikedFn != nil {
		return m.isLikedFn(postID, userID)
	}
	return false
}

func (m *mockService) GetDrafts(userID uint) ([]model.Post, error) {
	if m.getDraftsFn != nil {
		return m.getDraftsFn(userID)
	}
	return []model.Post{}, nil
}

// setupPostApp creates a fiber.App without auth middleware.
func setupPostApp(svc Service) *fiber.App {
	app := fiber.New()
	h := NewHandler(svc)

	app.Post("/posts", h.CreatePost)
	app.Get("/posts", h.GetPublicPosts)
	app.Get("/posts/:id", h.GetPost)
	app.Delete("/posts/:id", h.DeletePost)
	app.Post("/posts/:id/like", h.ToggleLike)
	app.Get("/posts/:id/like", h.GetLikeStatus)
	app.Get("/drafts", h.GetDrafts)

	return app
}

// setupAuthPostApp creates a fiber.App with userID injected via middleware.
func setupAuthPostApp(svc Service, userID uint) *fiber.App {
	app := fiber.New()
	h := NewHandler(svc)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	app.Post("/posts", h.CreatePost)
	app.Get("/posts", h.GetPublicPosts)
	app.Get("/posts/:id", h.GetPost)
	app.Delete("/posts/:id", h.DeletePost)
	app.Post("/posts/:id/like", h.ToggleLike)
	app.Get("/posts/:id/like", h.GetLikeStatus)
	app.Get("/drafts", h.GetDrafts)

	return app
}

// --- CreatePost ---

func TestCreatePost_Success(t *testing.T) {
	ms := &mockService{
		createPostFn: func(post *model.Post) error {
			post.ID = 1
			post.User = model.User{Username: "tester"}
			return nil
		},
	}
	app := setupAuthPostApp(ms, 1)

	body := `{"title":"Test Post","content":"Hello World","is_public":true}`
	req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
}

func TestCreatePost_Unauthorized(t *testing.T) {
	ms := &mockService{}
	app := setupPostApp(ms)

	body := `{"title":"Test Post","content":"Hello World"}`
	req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestCreatePost_InvalidJSON(t *testing.T) {
	ms := &mockService{}
	app := setupAuthPostApp(ms, 1)

	req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestCreatePost_ServiceError(t *testing.T) {
	ms := &mockService{
		createPostFn: func(post *model.Post) error {
			return errors.New("db error")
		},
	}
	app := setupAuthPostApp(ms, 1)

	body := `{"title":"Test Post","content":"Hello World"}`
	req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}

// --- GetPost ---

func TestGetPost_Success(t *testing.T) {
	ms := &mockService{
		getPostByIDFn: func(postID uint, userID *uint) (*model.Post, error) {
			return &model.Post{
				ID:       postID,
				Title:    "Test Post",
				Content:  "Hello World",
				IsPublic: true,
				User:     model.User{Username: "tester"},
			}, nil
		},
	}
	app := setupPostApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/posts/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
}

func TestGetPost_InvalidID(t *testing.T) {
	ms := &mockService{}
	app := setupPostApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/posts/abc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestGetPost_NotFound(t *testing.T) {
	ms := &mockService{
		getPostByIDFn: func(postID uint, userID *uint) (*model.Post, error) {
			return nil, ErrPostNotFound
		},
	}
	app := setupPostApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/posts/999", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}
}

func TestGetPost_Forbidden(t *testing.T) {
	ms := &mockService{
		getPostByIDFn: func(postID uint, userID *uint) (*model.Post, error) {
			return nil, ErrUnauthorized
		},
	}
	app := setupPostApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/posts/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", resp.StatusCode)
	}
}

func TestGetPost_ServiceError(t *testing.T) {
	ms := &mockService{
		getPostByIDFn: func(postID uint, userID *uint) (*model.Post, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupPostApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/posts/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}

// --- GetPublicPosts ---

func TestGetPublicPosts_Success(t *testing.T) {
	ms := &mockService{
		getPublicPostsFn: func(page, pageSize int, tag string) ([]model.Post, int64, error) {
			return []model.Post{
				{ID: 1, Title: "Post 1", IsPublic: true},
				{ID: 2, Title: "Post 2", IsPublic: true},
			}, 2, nil
		},
	}
	app := setupPostApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/posts?page=1&page_size=10", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
}

func TestGetPublicPosts_InvalidTag(t *testing.T) {
	ms := &mockService{
		getPublicPostsFn: func(page, pageSize int, tag string) ([]model.Post, int64, error) {
			return nil, 0, ErrInvalidTag
		},
	}
	app := setupPostApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/posts?tag=%invalid%", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestGetPublicPosts_ServiceError(t *testing.T) {
	ms := &mockService{
		getPublicPostsFn: func(page, pageSize int, tag string) ([]model.Post, int64, error) {
			return nil, 0, errors.New("db error")
		},
	}
	app := setupPostApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}

// --- DeletePost ---

func TestDeletePost_Success(t *testing.T) {
	ms := &mockService{
		deletePostFn: func(postID uint, userID uint) error {
			return nil
		},
	}
	app := setupAuthPostApp(ms, 1)

	req := httptest.NewRequest(http.MethodDelete, "/posts/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
}

func TestDeletePost_Unauthorized(t *testing.T) {
	ms := &mockService{}
	app := setupPostApp(ms)

	req := httptest.NewRequest(http.MethodDelete, "/posts/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestDeletePost_InvalidID(t *testing.T) {
	ms := &mockService{}
	app := setupAuthPostApp(ms, 1)

	req := httptest.NewRequest(http.MethodDelete, "/posts/abc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestDeletePost_NotFound(t *testing.T) {
	ms := &mockService{
		deletePostFn: func(postID uint, userID uint) error {
			return ErrPostNotFound
		},
	}
	app := setupAuthPostApp(ms, 1)

	req := httptest.NewRequest(http.MethodDelete, "/posts/999", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}
}

func TestDeletePost_Forbidden(t *testing.T) {
	ms := &mockService{
		deletePostFn: func(postID uint, userID uint) error {
			return ErrUnauthorized
		},
	}
	app := setupAuthPostApp(ms, 2)

	req := httptest.NewRequest(http.MethodDelete, "/posts/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", resp.StatusCode)
	}
}

func TestDeletePost_ServiceError(t *testing.T) {
	ms := &mockService{
		deletePostFn: func(postID uint, userID uint) error {
			return errors.New("db error")
		},
	}
	app := setupAuthPostApp(ms, 1)

	req := httptest.NewRequest(http.MethodDelete, "/posts/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}

// --- GetDrafts ---

func TestGetDrafts_Success(t *testing.T) {
	ms := &mockService{
		getDraftsFn: func(userID uint) ([]model.Post, error) {
			return []model.Post{
				{ID: 1, Title: "Draft 1", Status: "draft", UserID: userID},
			}, nil
		},
	}
	app := setupAuthPostApp(ms, 1)

	req := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
}

func TestGetDrafts_Unauthorized(t *testing.T) {
	ms := &mockService{}
	app := setupPostApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestGetDrafts_ServiceError(t *testing.T) {
	ms := &mockService{
		getDraftsFn: func(userID uint) ([]model.Post, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupAuthPostApp(ms, 1)

	req := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}

// --- ToggleLike ---

func TestToggleLike_Success(t *testing.T) {
	ms := &mockService{
		toggleLikeFn: func(postID uint, userID uint) (bool, uint, error) {
			return true, 1, nil
		},
	}
	app := setupAuthPostApp(ms, 1)

	req := httptest.NewRequest(http.MethodPost, "/posts/1/like", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
}

func TestToggleLike_Unauthorized(t *testing.T) {
	ms := &mockService{}
	app := setupPostApp(ms)

	req := httptest.NewRequest(http.MethodPost, "/posts/1/like", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestToggleLike_InvalidID(t *testing.T) {
	ms := &mockService{}
	app := setupAuthPostApp(ms, 1)

	req := httptest.NewRequest(http.MethodPost, "/posts/abc/like", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

// --- GetLikeStatus ---

func TestGetLikeStatus_Success(t *testing.T) {
	ms := &mockService{
		isLikedFn: func(postID uint, userID uint) bool {
			return true
		},
	}
	app := setupAuthPostApp(ms, 1)

	req := httptest.NewRequest(http.MethodGet, "/posts/1/like", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}
}

func TestGetLikeStatus_InvalidID(t *testing.T) {
	ms := &mockService{}
	app := setupPostApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/posts/abc/like", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestGetLikeStatus_NoAuth_ReturnsFalse(t *testing.T) {
	ms := &mockService{}
	app := setupPostApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/posts/1/like", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 (no auth means liked=false), got %d", resp.StatusCode)
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}
	liked, ok := data["liked"].(bool)
	if !ok || liked {
		t.Errorf("expected liked=false for unauthenticated user, got %v", data["liked"])
	}
}
