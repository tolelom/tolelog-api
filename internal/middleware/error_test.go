package middleware

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestErrorHandler_FiberError(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: ErrorHandler,
	})
	app.Get("/test", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusNotFound, "not found")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result["status"] != "error" {
		t.Errorf("expected status=error, got %s", result["status"])
	}
	if result["message"] != "not found" {
		t.Errorf("expected message=not found, got %s", result["message"])
	}
}

func TestErrorHandler_GenericError(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: ErrorHandler,
	})
	app.Get("/test", func(c *fiber.Ctx) error {
		return errors.New("something broke")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result["status"] != "error" {
		t.Errorf("expected status=error, got %s", result["status"])
	}
	// 500 에러는 내부 메시지를 숨겨야 함
	if result["message"] != "서버 내부 오류가 발생했습니다" {
		t.Errorf("expected masked message for 500, got %s", result["message"])
	}
}

func TestErrorHandler_ClientError(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: ErrorHandler,
	})
	app.Get("/test", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusBadRequest, "invalid input")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, _ := app.Test(req)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// 4xx 에러는 메시지를 그대로 전달
	if result["message"] != "invalid input" {
		t.Errorf("expected original message for 4xx, got %s", result["message"])
	}
}
