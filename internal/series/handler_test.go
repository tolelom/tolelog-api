package series

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

func setupTestApp(svc Service) *fiber.App {
	app := fiber.New()
	h := NewHandler(svc)

	app.Post("/series", h.CreateSeries)
	app.Get("/series/:id", h.GetSeries)
	app.Get("/users/:user_id/series", h.GetUserSeries)
	app.Put("/series/:id", h.UpdateSeries)
	app.Delete("/series/:id", h.DeleteSeries)
	app.Post("/series/:id/posts", h.AddPostToSeries)
	app.Delete("/series/:id/posts/:post_id", h.RemovePostFromSeries)
	app.Put("/series/:id/reorder", h.ReorderPosts)
	app.Get("/posts/:id/series-nav", h.GetSeriesNavigation)

	return app
}

func setupAuthApp(svc Service, userID uint) *fiber.App {
	app := fiber.New()
	h := NewHandler(svc)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	app.Post("/series", h.CreateSeries)
	app.Get("/series/:id", h.GetSeries)
	app.Get("/users/:user_id/series", h.GetUserSeries)
	app.Put("/series/:id", h.UpdateSeries)
	app.Delete("/series/:id", h.DeleteSeries)
	app.Post("/series/:id/posts", h.AddPostToSeries)
	app.Delete("/series/:id/posts/:post_id", h.RemovePostFromSeries)
	app.Put("/series/:id/reorder", h.ReorderPosts)
	app.Get("/posts/:id/series-nav", h.GetSeriesNavigation)

	return app
}

func TestNewHandler(t *testing.T) {
	h := NewHandler(&mockService{})
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

// --- CreateSeries ---

func TestCreateSeries_Handler_Success(t *testing.T) {
	ms := &mockService{
		createSeriesFn: func(series *model.Series) error {
			series.ID = 1
			series.User = model.User{Username: "tester"}
			return nil
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"title":"Go Basics","description":"Learn Go"}`
	req := httptest.NewRequest(http.MethodPost, "/series", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
}

func TestCreateSeries_Handler_Unauthorized(t *testing.T) {
	app := setupTestApp(&mockService{})

	body := `{"title":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/series", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestCreateSeries_Handler_InvalidJSON(t *testing.T) {
	app := setupAuthApp(&mockService{}, 1)

	req := httptest.NewRequest(http.MethodPost, "/series", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateSeries_Handler_EmptyTitle(t *testing.T) {
	app := setupAuthApp(&mockService{}, 1)

	body := `{"title":"","description":"desc"}`
	req := httptest.NewRequest(http.MethodPost, "/series", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateSeries_Handler_ServiceError(t *testing.T) {
	ms := &mockService{
		createSeriesFn: func(series *model.Series) error {
			return errors.New("db error")
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"title":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/series", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// --- GetSeries ---

func TestGetSeries_Handler_Success(t *testing.T) {
	ms := &mockService{
		getSeriesByIDFn: func(seriesID uint) (*model.Series, []model.Post, error) {
			s := &model.Series{ID: seriesID, Title: "Go", User: model.User{Username: "author"}}
			return s, []model.Post{}, nil
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/series/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGetSeries_Handler_InvalidID(t *testing.T) {
	app := setupTestApp(&mockService{})

	req := httptest.NewRequest(http.MethodGet, "/series/abc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetSeries_Handler_NotFound(t *testing.T) {
	ms := &mockService{
		getSeriesByIDFn: func(seriesID uint) (*model.Series, []model.Post, error) {
			return nil, nil, ErrSeriesNotFound
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/series/999", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestGetSeries_Handler_ServiceError(t *testing.T) {
	ms := &mockService{
		getSeriesByIDFn: func(seriesID uint) (*model.Series, []model.Post, error) {
			return nil, nil, errors.New("db error")
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/series/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// --- GetUserSeries ---

func TestGetUserSeries_Handler_Success(t *testing.T) {
	ms := &mockService{
		getSeriesByUserIDFn: func(userID uint) ([]model.Series, map[uint]int64, error) {
			list := []model.Series{
				{ID: 1, Title: "A", UserID: userID, User: model.User{Username: "u"}},
			}
			return list, map[uint]int64{1: 3}, nil
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/users/1/series", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("expected 'success', got %q", result.Status)
	}
}

func TestGetUserSeries_Handler_InvalidUserID(t *testing.T) {
	app := setupTestApp(&mockService{})

	req := httptest.NewRequest(http.MethodGet, "/users/abc/series", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetUserSeries_Handler_ServiceError(t *testing.T) {
	ms := &mockService{
		getSeriesByUserIDFn: func(userID uint) ([]model.Series, map[uint]int64, error) {
			return nil, nil, errors.New("db error")
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/users/1/series", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// --- UpdateSeries ---

func TestUpdateSeries_Handler_Success(t *testing.T) {
	ms := &mockService{
		updateSeriesFn: func(seriesID uint, userID uint, req *dto.UpdateSeriesRequest) (*model.Series, error) {
			return &model.Series{ID: seriesID, Title: "Updated", UserID: userID, User: model.User{Username: "u"}}, nil
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"title":"Updated"}`
	req := httptest.NewRequest(http.MethodPut, "/series/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestUpdateSeries_Handler_Unauthorized(t *testing.T) {
	app := setupTestApp(&mockService{})

	body := `{"title":"test"}`
	req := httptest.NewRequest(http.MethodPut, "/series/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestUpdateSeries_Handler_InvalidID(t *testing.T) {
	app := setupAuthApp(&mockService{}, 1)

	body := `{"title":"test"}`
	req := httptest.NewRequest(http.MethodPut, "/series/abc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestUpdateSeries_Handler_InvalidJSON(t *testing.T) {
	app := setupAuthApp(&mockService{}, 1)

	req := httptest.NewRequest(http.MethodPut, "/series/1", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestUpdateSeries_Handler_NotFound(t *testing.T) {
	ms := &mockService{
		updateSeriesFn: func(seriesID uint, userID uint, req *dto.UpdateSeriesRequest) (*model.Series, error) {
			return nil, ErrSeriesNotFound
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"title":"test"}`
	req := httptest.NewRequest(http.MethodPut, "/series/999", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateSeries_Handler_Forbidden(t *testing.T) {
	ms := &mockService{
		updateSeriesFn: func(seriesID uint, userID uint, req *dto.UpdateSeriesRequest) (*model.Series, error) {
			return nil, ErrUnauthorized
		},
	}
	app := setupAuthApp(ms, 2)

	body := `{"title":"test"}`
	req := httptest.NewRequest(http.MethodPut, "/series/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestUpdateSeries_Handler_ServiceError(t *testing.T) {
	ms := &mockService{
		updateSeriesFn: func(seriesID uint, userID uint, req *dto.UpdateSeriesRequest) (*model.Series, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"title":"test"}`
	req := httptest.NewRequest(http.MethodPut, "/series/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// --- DeleteSeries ---

func TestDeleteSeries_Handler_Success(t *testing.T) {
	ms := &mockService{
		deleteSeriesFn: func(seriesID uint, userID uint) error {
			return nil
		},
	}
	app := setupAuthApp(ms, 1)

	req := httptest.NewRequest(http.MethodDelete, "/series/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDeleteSeries_Handler_Unauthorized(t *testing.T) {
	app := setupTestApp(&mockService{})

	req := httptest.NewRequest(http.MethodDelete, "/series/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestDeleteSeries_Handler_InvalidID(t *testing.T) {
	app := setupAuthApp(&mockService{}, 1)

	req := httptest.NewRequest(http.MethodDelete, "/series/abc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestDeleteSeries_Handler_NotFound(t *testing.T) {
	ms := &mockService{
		deleteSeriesFn: func(seriesID uint, userID uint) error {
			return ErrSeriesNotFound
		},
	}
	app := setupAuthApp(ms, 1)

	req := httptest.NewRequest(http.MethodDelete, "/series/999", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteSeries_Handler_Forbidden(t *testing.T) {
	ms := &mockService{
		deleteSeriesFn: func(seriesID uint, userID uint) error {
			return ErrUnauthorized
		},
	}
	app := setupAuthApp(ms, 2)

	req := httptest.NewRequest(http.MethodDelete, "/series/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

// --- AddPostToSeries ---

func TestAddPostToSeries_Handler_Success(t *testing.T) {
	ms := &mockService{
		addPostToSeriesFn: func(seriesID uint, postID uint, order int, userID uint) error {
			return nil
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"post_id":10,"order":1}`
	req := httptest.NewRequest(http.MethodPost, "/series/1/posts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAddPostToSeries_Handler_Unauthorized(t *testing.T) {
	app := setupTestApp(&mockService{})

	body := `{"post_id":10,"order":1}`
	req := httptest.NewRequest(http.MethodPost, "/series/1/posts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAddPostToSeries_Handler_InvalidSeriesID(t *testing.T) {
	app := setupAuthApp(&mockService{}, 1)

	body := `{"post_id":10,"order":1}`
	req := httptest.NewRequest(http.MethodPost, "/series/abc/posts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestAddPostToSeries_Handler_InvalidJSON(t *testing.T) {
	app := setupAuthApp(&mockService{}, 1)

	req := httptest.NewRequest(http.MethodPost, "/series/1/posts", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestAddPostToSeries_Handler_SeriesNotFound(t *testing.T) {
	ms := &mockService{
		addPostToSeriesFn: func(seriesID uint, postID uint, order int, userID uint) error {
			return ErrSeriesNotFound
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"post_id":10,"order":1}`
	req := httptest.NewRequest(http.MethodPost, "/series/999/posts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestAddPostToSeries_Handler_PostNotFound(t *testing.T) {
	ms := &mockService{
		addPostToSeriesFn: func(seriesID uint, postID uint, order int, userID uint) error {
			return ErrPostNotFound
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"post_id":999,"order":1}`
	req := httptest.NewRequest(http.MethodPost, "/series/1/posts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestAddPostToSeries_Handler_PostNotOwned(t *testing.T) {
	ms := &mockService{
		addPostToSeriesFn: func(seriesID uint, postID uint, order int, userID uint) error {
			return ErrPostNotOwned
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"post_id":10,"order":1}`
	req := httptest.NewRequest(http.MethodPost, "/series/1/posts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestAddPostToSeries_Handler_Forbidden(t *testing.T) {
	ms := &mockService{
		addPostToSeriesFn: func(seriesID uint, postID uint, order int, userID uint) error {
			return ErrUnauthorized
		},
	}
	app := setupAuthApp(ms, 2)

	body := `{"post_id":10,"order":1}`
	req := httptest.NewRequest(http.MethodPost, "/series/1/posts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestAddPostToSeries_Handler_ServiceError(t *testing.T) {
	ms := &mockService{
		addPostToSeriesFn: func(seriesID uint, postID uint, order int, userID uint) error {
			return errors.New("db error")
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"post_id":10,"order":1}`
	req := httptest.NewRequest(http.MethodPost, "/series/1/posts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// --- RemovePostFromSeries ---

func TestRemovePostFromSeries_Handler_Success(t *testing.T) {
	ms := &mockService{
		removePostFromSeriesFn: func(seriesID uint, postID uint, userID uint) error {
			return nil
		},
	}
	app := setupAuthApp(ms, 1)

	req := httptest.NewRequest(http.MethodDelete, "/series/1/posts/10", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRemovePostFromSeries_Handler_Unauthorized(t *testing.T) {
	app := setupTestApp(&mockService{})

	req := httptest.NewRequest(http.MethodDelete, "/series/1/posts/10", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestRemovePostFromSeries_Handler_InvalidSeriesID(t *testing.T) {
	app := setupAuthApp(&mockService{}, 1)

	req := httptest.NewRequest(http.MethodDelete, "/series/abc/posts/10", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestRemovePostFromSeries_Handler_InvalidPostID(t *testing.T) {
	app := setupAuthApp(&mockService{}, 1)

	req := httptest.NewRequest(http.MethodDelete, "/series/1/posts/abc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestRemovePostFromSeries_Handler_SeriesNotFound(t *testing.T) {
	ms := &mockService{
		removePostFromSeriesFn: func(seriesID uint, postID uint, userID uint) error {
			return ErrSeriesNotFound
		},
	}
	app := setupAuthApp(ms, 1)

	req := httptest.NewRequest(http.MethodDelete, "/series/999/posts/10", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestRemovePostFromSeries_Handler_Forbidden(t *testing.T) {
	ms := &mockService{
		removePostFromSeriesFn: func(seriesID uint, postID uint, userID uint) error {
			return ErrUnauthorized
		},
	}
	app := setupAuthApp(ms, 2)

	req := httptest.NewRequest(http.MethodDelete, "/series/1/posts/10", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

// --- ReorderPosts ---

func TestReorderPosts_Handler_Success(t *testing.T) {
	ms := &mockService{
		reorderPostsFn: func(seriesID uint, postIDs []uint, userID uint) error {
			return nil
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"post_ids":[3,1,2]}`
	req := httptest.NewRequest(http.MethodPut, "/series/1/reorder", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestReorderPosts_Handler_Unauthorized(t *testing.T) {
	app := setupTestApp(&mockService{})

	body := `{"post_ids":[1,2]}`
	req := httptest.NewRequest(http.MethodPut, "/series/1/reorder", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestReorderPosts_Handler_InvalidSeriesID(t *testing.T) {
	app := setupAuthApp(&mockService{}, 1)

	body := `{"post_ids":[1]}`
	req := httptest.NewRequest(http.MethodPut, "/series/abc/reorder", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestReorderPosts_Handler_InvalidJSON(t *testing.T) {
	app := setupAuthApp(&mockService{}, 1)

	req := httptest.NewRequest(http.MethodPut, "/series/1/reorder", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestReorderPosts_Handler_SeriesNotFound(t *testing.T) {
	ms := &mockService{
		reorderPostsFn: func(seriesID uint, postIDs []uint, userID uint) error {
			return ErrSeriesNotFound
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"post_ids":[1]}`
	req := httptest.NewRequest(http.MethodPut, "/series/999/reorder", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestReorderPosts_Handler_Forbidden(t *testing.T) {
	ms := &mockService{
		reorderPostsFn: func(seriesID uint, postIDs []uint, userID uint) error {
			return ErrUnauthorized
		},
	}
	app := setupAuthApp(ms, 2)

	body := `{"post_ids":[1]}`
	req := httptest.NewRequest(http.MethodPut, "/series/1/reorder", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestReorderPosts_Handler_ServiceError(t *testing.T) {
	ms := &mockService{
		reorderPostsFn: func(seriesID uint, postIDs []uint, userID uint) error {
			return errors.New("db error")
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"post_ids":[1]}`
	req := httptest.NewRequest(http.MethodPut, "/series/1/reorder", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// --- GetSeriesNavigation ---

func TestGetSeriesNavigation_Handler_Success(t *testing.T) {
	ms := &mockService{
		getSeriesNavigationFn: func(postID uint) (*dto.SeriesNavResponse, error) {
			return &dto.SeriesNavResponse{
				SeriesID:    1,
				SeriesTitle: "Go Basics",
				TotalPosts:  3,
			}, nil
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/posts/1/series-nav", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGetSeriesNavigation_Handler_NilNavigation(t *testing.T) {
	ms := &mockService{
		getSeriesNavigationFn: func(postID uint) (*dto.SeriesNavResponse, error) {
			return nil, nil
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/posts/1/series-nav", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if result.Data != nil {
		t.Errorf("expected nil data for post not in series, got %v", result.Data)
	}
}

func TestGetSeriesNavigation_Handler_InvalidID(t *testing.T) {
	app := setupTestApp(&mockService{})

	req := httptest.NewRequest(http.MethodGet, "/posts/abc/series-nav", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetSeriesNavigation_Handler_ServiceError(t *testing.T) {
	ms := &mockService{
		getSeriesNavigationFn: func(postID uint) (*dto.SeriesNavResponse, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/posts/1/series-nav", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}
