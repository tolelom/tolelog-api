package image

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"tolelom_api/internal/dto"

	"github.com/gofiber/fiber/v2"
)

func setupTestApp(uploadDir string) *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024 * 1024, // 10MB to test our own size validation
	})
	h := NewHandler(uploadDir)
	app.Post("/upload", h.Upload)
	return app
}

// createMultipartBody builds a multipart form body with the given field name, filename, and content.
func createMultipartBody(fieldName, filename string, content []byte) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile(fieldName, filename)
	_, _ = part.Write(content)
	_ = writer.Close()
	return body, writer.FormDataContentType()
}

// createSmallPNG generates a minimal valid PNG image.
func createSmallPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	buf := &bytes.Buffer{}
	_ = png.Encode(buf, img)
	return buf.Bytes()
}

// createSmallJPEG generates minimal JPEG-like bytes (JPEG magic bytes + padding).
func createSmallJPEG() []byte {
	// JFIF magic: FF D8 FF E0
	data := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	// Pad to 512 bytes so DetectContentType recognizes it
	padding := make([]byte, 508)
	return append(data, padding...)
}

func TestNewHandler(t *testing.T) {
	h := NewHandler("/tmp")
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestUpload_Handler_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "image-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	app := setupTestApp(tmpDir)

	pngData := createSmallPNG()
	body, contentType := createMultipartBody("image", "test.png", pngData)

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("expected 'success', got %q", result.Status)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}
	url, ok := data["url"].(string)
	if !ok || url == "" {
		t.Error("expected non-empty url in response")
	}

	// Verify file was actually saved
	files, _ := filepath.Glob(filepath.Join(tmpDir, "*.png"))
	if len(files) == 0 {
		t.Error("expected file to be saved in upload directory")
	}
}

func TestUpload_Handler_NoFile(t *testing.T) {
	app := setupTestApp("/tmp")

	req := httptest.NewRequest(http.MethodPost, "/upload", nil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}

	var errResp dto.ErrorResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp.Error != "no_file" {
		t.Errorf("expected error code 'no_file', got %q", errResp.Error)
	}
}

func TestUpload_Handler_WrongFieldName(t *testing.T) {
	app := setupTestApp("/tmp")

	pngData := createSmallPNG()
	body, contentType := createMultipartBody("wrong_field", "test.png", pngData)

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for wrong field name, got %d", resp.StatusCode)
	}
}

func TestUpload_Handler_InvalidMIMEType(t *testing.T) {
	app := setupTestApp("/tmp")

	// Send plain text as file — should be rejected
	textContent := []byte("this is not an image")
	body, contentType := createMultipartBody("image", "test.txt", textContent)

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid MIME type, got %d", resp.StatusCode)
	}

	var errResp dto.ErrorResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp.Error != "invalid_file" {
		t.Errorf("expected error code 'invalid_file', got %q", errResp.Error)
	}
}

func TestUpload_Handler_FileTooLarge(t *testing.T) {
	app := setupTestApp("/tmp")

	// Create a file > 5MB (PNG header + large padding)
	pngData := createSmallPNG()
	// Pad to exceed 5MB
	largeData := make([]byte, 6*1024*1024)
	copy(largeData, pngData)

	body, contentType := createMultipartBody("image", "large.png", largeData)

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for file too large, got %d", resp.StatusCode)
	}
}

func TestUpload_Handler_SaveToInvalidDir(t *testing.T) {
	// Use a non-existent directory to trigger save failure
	app := setupTestApp("/nonexistent/path/that/does/not/exist")

	pngData := createSmallPNG()
	body, contentType := createMultipartBody("image", "test.png", pngData)

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 for save failure, got %d", resp.StatusCode)
	}

	var errResp dto.ErrorResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp.Error != "upload_failed" {
		t.Errorf("expected error code 'upload_failed', got %q", errResp.Error)
	}
}

func TestUpload_Handler_JPEGFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "image-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	app := setupTestApp(tmpDir)

	jpegData := createSmallJPEG()
	body, contentType := createMultipartBody("image", "photo.jpg", jpegData)

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Errorf("expected 200 for JPEG, got %d: %s", resp.StatusCode, string(respBody))
	}
}

func TestUpload_Handler_ResponseURLFormat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "image-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	app := setupTestApp(tmpDir)

	pngData := createSmallPNG()
	body, contentType := createMultipartBody("image", "test.png", pngData)

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	var result dto.SuccessResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	data := result.Data.(map[string]interface{})
	url := data["url"].(string)

	// URL should start with /uploads/images/ and end with .png
	if len(url) < 20 {
		t.Errorf("URL seems too short: %q", url)
	}
	prefix := "/uploads/images/"
	if len(url) < len(prefix) || url[:len(prefix)] != prefix {
		t.Errorf("expected URL to start with '/uploads/images/', got %q", url)
	}
	if filepath.Ext(url) != ".png" {
		t.Errorf("expected .png extension, got %q", filepath.Ext(url))
	}
}
