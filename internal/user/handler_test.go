package user

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

func setupTestApp(svc AuthService) *fiber.App {
	app := fiber.New()
	h := NewHandler(svc, "/tmp/uploads")

	app.Post("/auth/register", h.Register)
	app.Post("/auth/login", h.Login)
	app.Post("/auth/refresh", h.RefreshToken)
	app.Get("/users/:user_id", h.GetProfile)
	app.Put("/users/avatar", h.UploadAvatar)
	app.Put("/users/password", h.ChangePassword)

	return app
}

func setupAuthApp(svc AuthService, userID uint) *fiber.App {
	app := fiber.New()
	h := NewHandler(svc, "/tmp/uploads")

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	app.Post("/auth/register", h.Register)
	app.Post("/auth/login", h.Login)
	app.Post("/auth/refresh", h.RefreshToken)
	app.Get("/users/:user_id", h.GetProfile)
	app.Put("/users/avatar", h.UploadAvatar)
	app.Put("/users/password", h.ChangePassword)

	return app
}

func newAuthResponse() *dto.AuthResponse {
	return &dto.AuthResponse{
		User:         dto.UserResponse{ID: 1, Username: "testuser"},
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
	}
}

func TestNewHandler(t *testing.T) {
	h := NewHandler(&mockAuthService{}, "/tmp")
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

// --- Register ---

func TestRegister_Handler_Success(t *testing.T) {
	ms := &mockAuthService{
		registerUserFn: func(req *dto.RegisterRequest) (*dto.AuthResponse, error) {
			return newAuthResponse(), nil
		},
	}
	app := setupTestApp(ms)

	body := `{"username":"test_user","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
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

func TestRegister_Handler_InvalidJSON(t *testing.T) {
	app := setupTestApp(&mockAuthService{})

	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestRegister_Handler_ShortUsername(t *testing.T) {
	app := setupTestApp(&mockAuthService{})

	body := `{"username":"ab","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for short username, got %d", resp.StatusCode)
	}
}

func TestRegister_Handler_ShortPassword(t *testing.T) {
	app := setupTestApp(&mockAuthService{})

	body := `{"username":"test_user","password":"12345"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for short password, got %d", resp.StatusCode)
	}
}

func TestRegister_Handler_MissingFields(t *testing.T) {
	app := setupTestApp(&mockAuthService{})

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestRegister_Handler_UserAlreadyExists(t *testing.T) {
	ms := &mockAuthService{
		registerUserFn: func(req *dto.RegisterRequest) (*dto.AuthResponse, error) {
			return nil, ErrUserAlreadyExists
		},
	}
	app := setupTestApp(ms)

	body := `{"username":"test_user","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
}

func TestRegister_Handler_ServiceError(t *testing.T) {
	ms := &mockAuthService{
		registerUserFn: func(req *dto.RegisterRequest) (*dto.AuthResponse, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupTestApp(ms)

	body := `{"username":"test_user","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// --- Login ---

func TestLogin_Handler_Success(t *testing.T) {
	ms := &mockAuthService{
		authenticateUserFn: func(req *dto.LoginRequest) (*dto.AuthResponse, error) {
			return newAuthResponse(), nil
		},
	}
	app := setupTestApp(ms)

	body := `{"username":"test_user","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestLogin_Handler_InvalidJSON(t *testing.T) {
	app := setupTestApp(&mockAuthService{})

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestLogin_Handler_MissingFields(t *testing.T) {
	app := setupTestApp(&mockAuthService{})

	body := `{"username":"test_user"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestLogin_Handler_InvalidCredentials(t *testing.T) {
	ms := &mockAuthService{
		authenticateUserFn: func(req *dto.LoginRequest) (*dto.AuthResponse, error) {
			return nil, ErrInvalidCredentials
		},
	}
	app := setupTestApp(ms)

	body := `{"username":"test_user","password":"wrongpass"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestLogin_Handler_ServiceError(t *testing.T) {
	ms := &mockAuthService{
		authenticateUserFn: func(req *dto.LoginRequest) (*dto.AuthResponse, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupTestApp(ms)

	body := `{"username":"test_user","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// --- RefreshToken ---

func TestRefreshToken_Handler_Success(t *testing.T) {
	ms := &mockAuthService{
		refreshTokensFn: func(refreshToken string) (*dto.AuthResponse, error) {
			return newAuthResponse(), nil
		},
	}
	app := setupTestApp(ms)

	body := `{"refresh_token":"valid-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRefreshToken_Handler_InvalidJSON(t *testing.T) {
	app := setupTestApp(&mockAuthService{})

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestRefreshToken_Handler_MissingToken(t *testing.T) {
	app := setupTestApp(&mockAuthService{})

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestRefreshToken_Handler_InvalidToken(t *testing.T) {
	ms := &mockAuthService{
		refreshTokensFn: func(refreshToken string) (*dto.AuthResponse, error) {
			return nil, errors.New("invalid token")
		},
	}
	app := setupTestApp(ms)

	body := `{"refresh_token":"bad-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

// --- GetProfile ---

func TestGetProfile_Handler_Success(t *testing.T) {
	ms := &mockAuthService{
		getUserByIDFn: func(userID uint) (*model.User, error) {
			return &model.User{ID: userID, Username: "testuser"}, nil
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
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

func TestGetProfile_Handler_InvalidID(t *testing.T) {
	app := setupTestApp(&mockAuthService{})

	req := httptest.NewRequest(http.MethodGet, "/users/abc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetProfile_Handler_NotFound(t *testing.T) {
	ms := &mockAuthService{
		getUserByIDFn: func(userID uint) (*model.User, error) {
			return nil, ErrUserNotFound
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/users/999", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestGetProfile_Handler_ServiceError(t *testing.T) {
	ms := &mockAuthService{
		getUserByIDFn: func(userID uint) (*model.User, error) {
			return nil, errors.New("db error")
		},
	}
	app := setupTestApp(ms)

	req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// --- UploadAvatar ---

func TestUploadAvatar_Handler_Unauthorized(t *testing.T) {
	app := setupTestApp(&mockAuthService{})

	req := httptest.NewRequest(http.MethodPut, "/users/avatar", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestUploadAvatar_Handler_NoFile(t *testing.T) {
	app := setupAuthApp(&mockAuthService{}, 1)

	req := httptest.NewRequest(http.MethodPut, "/users/avatar", nil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// --- Error response format ---

func TestRegister_Handler_ErrorFormat(t *testing.T) {
	ms := &mockAuthService{
		registerUserFn: func(req *dto.RegisterRequest) (*dto.AuthResponse, error) {
			return nil, ErrUserAlreadyExists
		},
	}
	app := setupTestApp(ms)

	body := `{"username":"test_user","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	var errResp dto.ErrorResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp.Status != "error" {
		t.Errorf("expected status 'error', got %q", errResp.Status)
	}
	if errResp.Error != "user_already_exists" {
		t.Errorf("expected error code 'user_already_exists', got %q", errResp.Error)
	}
}

func TestLogin_Handler_ErrorFormat(t *testing.T) {
	ms := &mockAuthService{
		authenticateUserFn: func(req *dto.LoginRequest) (*dto.AuthResponse, error) {
			return nil, ErrInvalidCredentials
		},
	}
	app := setupTestApp(ms)

	body := `{"username":"test_user","password":"wrongpass"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	var errResp dto.ErrorResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp.Error != "invalid_credentials" {
		t.Errorf("expected 'invalid_credentials', got %q", errResp.Error)
	}
}

// --- Username validation ---

func TestRegister_Handler_InvalidUsername(t *testing.T) {
	app := setupTestApp(&mockAuthService{})

	// Username with special chars should fail alphanum_underscore validation
	body := `{"username":"test-user!","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid username, got %d", resp.StatusCode)
	}
}

// setupLogoutApp creates a fiber.App with Logout and DeleteMe routes.
func setupLogoutApp(svc AuthService) *fiber.App {
	app := fiber.New()
	h := NewHandler(svc, "/tmp/uploads")

	app.Post("/auth/logout", h.Logout)
	app.Delete("/users/me", h.DeleteMe)

	return app
}

// setupAuthLogoutApp creates a fiber.App with auth middleware plus Logout and DeleteMe routes.
func setupAuthLogoutApp(svc AuthService, userID uint) *fiber.App {
	app := fiber.New()
	h := NewHandler(svc, "/tmp/uploads")

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	app.Post("/auth/logout", h.Logout)
	app.Delete("/users/me", h.DeleteMe)

	return app
}

// --- Logout ---

func TestLogout_Success(t *testing.T) {
	ms := &mockAuthService{
		logoutFn: func(userID uint) error {
			return nil
		},
	}
	app := setupAuthLogoutApp(ms, 1)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
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

func TestLogout_Unauthorized(t *testing.T) {
	ms := &mockAuthService{}
	app := setupLogoutApp(ms)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestLogout_ServiceError(t *testing.T) {
	ms := &mockAuthService{
		logoutFn: func(userID uint) error {
			return errors.New("logout db error")
		},
	}
	app := setupAuthLogoutApp(ms, 1)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// --- DeleteMe ---

func TestDeleteMe_Success(t *testing.T) {
	ms := &mockAuthService{
		deleteUserFn: func(userID uint) error {
			return nil
		},
	}
	app := setupAuthLogoutApp(ms, 1)

	req := httptest.NewRequest(http.MethodDelete, "/users/me", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
}

func TestDeleteMe_Unauthorized(t *testing.T) {
	ms := &mockAuthService{}
	app := setupLogoutApp(ms)

	req := httptest.NewRequest(http.MethodDelete, "/users/me", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestDeleteMe_ServiceError(t *testing.T) {
	ms := &mockAuthService{
		deleteUserFn: func(userID uint) error {
			return errors.New("delete db error")
		},
	}
	app := setupAuthLogoutApp(ms, 1)

	req := httptest.NewRequest(http.MethodDelete, "/users/me", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// --- ChangePassword ---

func TestChangePassword_Success(t *testing.T) {
	ms := &mockAuthService{
		changePasswordFn: func(userID uint, req *dto.ChangePasswordRequest) error {
			return nil
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"current_password":"oldpass1","new_password":"newpass1"}`
	req := httptest.NewRequest(http.MethodPut, "/users/password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

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

func TestChangePassword_Unauthorized(t *testing.T) {
	app := setupTestApp(&mockAuthService{})

	body := `{"current_password":"oldpass1","new_password":"newpass1"}`
	req := httptest.NewRequest(http.MethodPut, "/users/password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestChangePassword_InvalidJSON(t *testing.T) {
	app := setupAuthApp(&mockAuthService{}, 1)

	req := httptest.NewRequest(http.MethodPut, "/users/password", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestChangePassword_MissingFields(t *testing.T) {
	app := setupAuthApp(&mockAuthService{}, 1)

	body := `{"current_password":"oldpass1"}`
	req := httptest.NewRequest(http.MethodPut, "/users/password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestChangePassword_NewPasswordTooShort(t *testing.T) {
	app := setupAuthApp(&mockAuthService{}, 1)

	body := `{"current_password":"oldpass1","new_password":"ab"}`
	req := httptest.NewRequest(http.MethodPut, "/users/password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestChangePassword_InvalidCurrentPassword(t *testing.T) {
	ms := &mockAuthService{
		changePasswordFn: func(userID uint, req *dto.ChangePasswordRequest) error {
			return ErrInvalidPassword
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"current_password":"wrongpass","new_password":"newpass1"}`
	req := httptest.NewRequest(http.MethodPut, "/users/password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestChangePassword_SamePassword(t *testing.T) {
	ms := &mockAuthService{
		changePasswordFn: func(userID uint, req *dto.ChangePasswordRequest) error {
			return ErrSamePassword
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"current_password":"samepass1","new_password":"samepass1"}`
	req := httptest.NewRequest(http.MethodPut, "/users/password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestChangePassword_ServiceError(t *testing.T) {
	ms := &mockAuthService{
		changePasswordFn: func(userID uint, req *dto.ChangePasswordRequest) error {
			return errors.New("db error")
		},
	}
	app := setupAuthApp(ms, 1)

	body := `{"current_password":"oldpass1","new_password":"newpass1"}`
	req := httptest.NewRequest(http.MethodPut, "/users/password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}
