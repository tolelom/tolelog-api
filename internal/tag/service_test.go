package tag

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"tolelom_api/internal/dto"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// mockService implements Service for testing.
type mockService struct {
	getTagsFn func(sort string, limit int) (*dto.TagListResponse, error)
}

func (m *mockService) GetTags(sort string, limit int) (*dto.TagListResponse, error) {
	if m.getTagsFn != nil {
		return m.getTagsFn(sort, limit)
	}
	return &dto.TagListResponse{Tags: []dto.TagResponse{}, Total: 0}, nil
}

// --- Service unit tests ---

func TestNewService(t *testing.T) {
	db := &gorm.DB{}
	svc := NewService(db)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestMockService_GetTags_Success(t *testing.T) {
	ms := &mockService{
		getTagsFn: func(sort string, limit int) (*dto.TagListResponse, error) {
			return &dto.TagListResponse{
				Tags:  []dto.TagResponse{{Name: "go", PostCount: 5}},
				Total: 1,
			}, nil
		},
	}
	result, err := ms.GetTags("popular", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total 1, got %d", result.Total)
	}
	if len(result.Tags) != 1 || result.Tags[0].Name != "go" {
		t.Errorf("unexpected tags: %v", result.Tags)
	}
}

func TestMockService_GetTags_DBError(t *testing.T) {
	dbErr := errors.New("db error")
	ms := &mockService{
		getTagsFn: func(sort string, limit int) (*dto.TagListResponse, error) {
			return nil, dbErr
		},
	}
	_, err := ms.GetTags("popular", 50)
	if !errors.Is(err, dbErr) {
		t.Errorf("expected db error, got %v", err)
	}
}

func TestMockService_GetTags_Empty(t *testing.T) {
	ms := &mockService{
		getTagsFn: func(sort string, limit int) (*dto.TagListResponse, error) {
			return &dto.TagListResponse{Tags: []dto.TagResponse{}, Total: 0}, nil
		},
	}
	result, err := ms.GetTags("name", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 0 || len(result.Tags) != 0 {
		t.Errorf("expected empty result, got %+v", result)
	}
}

// --- Handler tests ---

func setupTestApp(svc Service) *fiber.App {
	app := fiber.New()
	h := NewHandler(svc)
	app.Get("/tags", h.GetTags)
	return app
}

func TestNewHandler(t *testing.T) {
	h := NewHandler(&mockService{})
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestGetTags_Handler_Success(t *testing.T) {
	ms := &mockService{
		getTagsFn: func(sort string, limit int) (*dto.TagListResponse, error) {
			return &dto.TagListResponse{
				Tags:  []dto.TagResponse{{Name: "go", PostCount: 5}},
				Total: 1,
			}, nil
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/tags", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGetTags_Handler_SortPopular(t *testing.T) {
	var capturedSort string
	ms := &mockService{
		getTagsFn: func(sort string, limit int) (*dto.TagListResponse, error) {
			capturedSort = sort
			return &dto.TagListResponse{Tags: []dto.TagResponse{}, Total: 0}, nil
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/tags?sort=popular", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if capturedSort != "popular" {
		t.Errorf("expected sort=popular, got %q", capturedSort)
	}
}

func TestGetTags_Handler_SortName(t *testing.T) {
	var capturedSort string
	ms := &mockService{
		getTagsFn: func(sort string, limit int) (*dto.TagListResponse, error) {
			capturedSort = sort
			return &dto.TagListResponse{Tags: []dto.TagResponse{}, Total: 0}, nil
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/tags?sort=name", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if capturedSort != "name" {
		t.Errorf("expected sort=name, got %q", capturedSort)
	}
}

func TestGetTags_Handler_InvalidSortFallsToPopular(t *testing.T) {
	var capturedSort string
	ms := &mockService{
		getTagsFn: func(sort string, limit int) (*dto.TagListResponse, error) {
			capturedSort = sort
			return &dto.TagListResponse{Tags: []dto.TagResponse{}, Total: 0}, nil
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/tags?sort=invalid", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if capturedSort != "popular" {
		t.Errorf("expected sort fallback to popular, got %q", capturedSort)
	}
}

func TestGetTags_Handler_LimitParam(t *testing.T) {
	var capturedLimit int
	ms := &mockService{
		getTagsFn: func(sort string, limit int) (*dto.TagListResponse, error) {
			capturedLimit = limit
			return &dto.TagListResponse{Tags: []dto.TagResponse{}, Total: 0}, nil
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/tags?limit=100", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if capturedLimit != 100 {
		t.Errorf("expected limit 100, got %d", capturedLimit)
	}
}

func TestGetTags_Handler_LimitTooLargeFallsToDefault(t *testing.T) {
	var capturedLimit int
	ms := &mockService{
		getTagsFn: func(sort string, limit int) (*dto.TagListResponse, error) {
			capturedLimit = limit
			return &dto.TagListResponse{Tags: []dto.TagResponse{}, Total: 0}, nil
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/tags?limit=9999", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if capturedLimit != 50 {
		t.Errorf("expected fallback limit 50, got %d", capturedLimit)
	}
}

func TestGetTags_Handler_LimitZeroFallsToDefault(t *testing.T) {
	var capturedLimit int
	ms := &mockService{
		getTagsFn: func(sort string, limit int) (*dto.TagListResponse, error) {
			capturedLimit = limit
			return &dto.TagListResponse{Tags: []dto.TagResponse{}, Total: 0}, nil
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/tags?limit=0", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if capturedLimit != 50 {
		t.Errorf("expected fallback limit 50, got %d", capturedLimit)
	}
}

func TestGetTags_Handler_ServiceError(t *testing.T) {
	ms := &mockService{
		getTagsFn: func(sort string, limit int) (*dto.TagListResponse, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/tags", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}
