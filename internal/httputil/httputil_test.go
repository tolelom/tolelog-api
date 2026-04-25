package httputil

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// --- ParsePagination ---

func TestParsePagination_Defaults(t *testing.T) {
	app := fiber.New()
	var gotPage, gotPageSize int
	app.Get("/", func(c *fiber.Ctx) error {
		gotPage, gotPageSize = ParsePagination(c)
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	app.Test(req) //nolint:errcheck

	if gotPage != 1 {
		t.Errorf("expected default page=1, got %d", gotPage)
	}
	if gotPageSize != 10 {
		t.Errorf("expected default pageSize=10, got %d", gotPageSize)
	}
}

func TestParsePagination_Custom(t *testing.T) {
	app := fiber.New()
	var gotPage, gotPageSize int
	app.Get("/", func(c *fiber.Ctx) error {
		gotPage, gotPageSize = ParsePagination(c)
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/?page=3&page_size=25", nil)
	app.Test(req) //nolint:errcheck

	if gotPage != 3 {
		t.Errorf("expected page=3, got %d", gotPage)
	}
	if gotPageSize != 25 {
		t.Errorf("expected pageSize=25, got %d", gotPageSize)
	}
}

func TestParsePagination_PageBelowMin(t *testing.T) {
	app := fiber.New()
	var gotPage, _ int
	app.Get("/", func(c *fiber.Ctx) error {
		gotPage, _ = ParsePagination(c)
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/?page=0", nil)
	app.Test(req) //nolint:errcheck

	if gotPage != 1 {
		t.Errorf("expected page clamped to 1, got %d", gotPage)
	}
}

func TestParsePagination_PageSizeTooLarge(t *testing.T) {
	app := fiber.New()
	var _, gotPageSize int
	app.Get("/", func(c *fiber.Ctx) error {
		_, gotPageSize = ParsePagination(c)
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/?page_size=500", nil)
	app.Test(req) //nolint:errcheck

	if gotPageSize != 10 {
		t.Errorf("expected pageSize clamped to 10, got %d", gotPageSize)
	}
}

func TestParsePagination_PageSizeZero(t *testing.T) {
	app := fiber.New()
	var _, gotPageSize int
	app.Get("/", func(c *fiber.Ctx) error {
		_, gotPageSize = ParsePagination(c)
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/?page_size=0", nil)
	app.Test(req) //nolint:errcheck

	if gotPageSize != 10 {
		t.Errorf("expected pageSize clamped to 10, got %d", gotPageSize)
	}
}

// --- RequireAuth ---

func TestRequireAuth_Success(t *testing.T) {
	app := fiber.New()
	var gotID uint
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", uint(42))
		return c.Next()
	})
	app.Get("/", func(c *fiber.Ctx) error {
		id, err := RequireAuth(c)
		if err != nil {
			return err
		}
		gotID = id
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if gotID != 42 {
		t.Errorf("expected userID=42, got %d", gotID)
	}
}

func TestRequireAuth_NoAuth(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		_, err := RequireAuth(c)
		if err != nil {
			return nil
		}
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

// --- BindAndValidate ---

type testRequest struct {
	Name string `json:"name" validate:"required,min=2"`
}

func TestBindAndValidate_Success(t *testing.T) {
	app := fiber.New()
	app.Post("/", func(c *fiber.Ctx) error {
		req, err := BindAndValidate[testRequest](c)
		if err != nil {
			return nil
		}
		return c.JSON(req)
	})

	body := `{"name":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestBindAndValidate_InvalidJSON(t *testing.T) {
	app := fiber.New()
	app.Post("/", func(c *fiber.Ctx) error {
		_, err := BindAndValidate[testRequest](c)
		if err != nil {
			return nil
		}
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestBindAndValidate_ValidationFailure(t *testing.T) {
	app := fiber.New()
	app.Post("/", func(c *fiber.Ctx) error {
		_, err := BindAndValidate[testRequest](c)
		if err != nil {
			return nil
		}
		return c.SendStatus(fiber.StatusOK)
	})

	body := `{"name":"a"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// --- NewPagination ---

func TestNewPagination_TotalPages(t *testing.T) {
	tests := []struct {
		page, pageSize int
		total          int64
		wantPages      int64
	}{
		{1, 10, 0, 0},
		{1, 10, 10, 1},
		{1, 10, 11, 2},
		{1, 10, 100, 10},
		{1, 10, 101, 11},
		{1, 3, 7, 3},
	}

	for _, tt := range tests {
		p := NewPagination(tt.page, tt.pageSize, tt.total)
		if p.TotalPages != tt.wantPages {
			t.Errorf("NewPagination(page=%d, size=%d, total=%d).TotalPages = %d, want %d",
				tt.page, tt.pageSize, tt.total, p.TotalPages, tt.wantPages)
		}
		if p.Page != tt.page {
			t.Errorf("expected page=%d, got %d", tt.page, p.Page)
		}
		if p.PageSize != tt.pageSize {
			t.Errorf("expected pageSize=%d, got %d", tt.pageSize, p.PageSize)
		}
		if p.Total != tt.total {
			t.Errorf("expected total=%d, got %d", tt.total, p.Total)
		}
	}
}
