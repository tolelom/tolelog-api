package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"tolelom_api/internal/config"
	"tolelom_api/internal/model"
	"tolelom_api/internal/utils"

	"github.com/gofiber/fiber/v2"
)

const testJWTSecret = "test-secret-for-auth-middleware"

func setupTestApp(handler fiber.Handler) *fiber.App {
	app := fiber.New()
	cfg := &config.Config{JWTSecret: testJWTSecret}

	app.Get("/protected", AuthMiddleware(cfg), handler)
	app.Get("/optional", OptionalAuthMiddleware(cfg), handler)
	return app
}

func generateTestToken(userID uint, username string) string {
	user := &model.User{ID: userID, Username: username}
	token, _, _ := utils.GenerateTokenPair(user, testJWTSecret)
	return token
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	var capturedUserID uint
	handler := func(c *fiber.Ctx) error {
		capturedUserID = c.Locals("userID").(uint)
		return c.SendStatus(fiber.StatusOK)
	}

	app := setupTestApp(handler)
	token := generateTestToken(42, "testuser")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if capturedUserID != 42 {
		t.Errorf("expected userID=42, got %d", capturedUserID)
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	app := setupTestApp(func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	app := setupTestApp(func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "InvalidFormat token123")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	app := setupTestApp(func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-jwt-token")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestOptionalAuthMiddleware_NoToken(t *testing.T) {
	var hasUserID bool
	handler := func(c *fiber.Ctx) error {
		_, hasUserID = c.Locals("userID").(uint)
		return c.SendStatus(fiber.StatusOK)
	}

	app := setupTestApp(handler)

	req := httptest.NewRequest(http.MethodGet, "/optional", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if hasUserID {
		t.Error("expected no userID in context when no token provided")
	}
}

func TestOptionalAuthMiddleware_ValidToken(t *testing.T) {
	var capturedUserID uint
	handler := func(c *fiber.Ctx) error {
		capturedUserID = c.Locals("userID").(uint)
		return c.SendStatus(fiber.StatusOK)
	}

	app := setupTestApp(handler)
	token := generateTestToken(99, "optuser")

	req := httptest.NewRequest(http.MethodGet, "/optional", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if capturedUserID != 99 {
		t.Errorf("expected userID=99, got %d", capturedUserID)
	}
}

func TestOptionalAuthMiddleware_InvalidToken(t *testing.T) {
	var hasUserID bool
	handler := func(c *fiber.Ctx) error {
		_, hasUserID = c.Locals("userID").(uint)
		return c.SendStatus(fiber.StatusOK)
	}

	app := setupTestApp(handler)

	req := httptest.NewRequest(http.MethodGet, "/optional", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200 (pass through), got %d", resp.StatusCode)
	}
	if hasUserID {
		t.Error("expected no userID for invalid token")
	}
}
