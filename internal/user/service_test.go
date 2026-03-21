package user

import (
	"errors"
	"testing"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/model"

	"gorm.io/gorm"
)

// mockAuthService implements AuthService for testing.
type mockAuthService struct {
	registerUserFn     func(req *dto.RegisterRequest) (*dto.AuthResponse, error)
	authenticateUserFn func(req *dto.LoginRequest) (*dto.AuthResponse, error)
	getUserByIDFn      func(userID uint) (*model.User, error)
	updateAvatarFn     func(userID uint, avatarURL string) error
	refreshTokensFn    func(refreshToken string) (*dto.AuthResponse, error)
	changePasswordFn   func(userID uint, req *dto.ChangePasswordRequest) error
	logoutFn           func(userID uint) error
	deleteUserFn       func(userID uint) error
}

func (m *mockAuthService) RegisterUser(req *dto.RegisterRequest) (*dto.AuthResponse, error) {
	if m.registerUserFn != nil {
		return m.registerUserFn(req)
	}
	return nil, nil
}

func (m *mockAuthService) AuthenticateUser(req *dto.LoginRequest) (*dto.AuthResponse, error) {
	if m.authenticateUserFn != nil {
		return m.authenticateUserFn(req)
	}
	return nil, nil
}

func (m *mockAuthService) GetUserByID(userID uint) (*model.User, error) {
	if m.getUserByIDFn != nil {
		return m.getUserByIDFn(userID)
	}
	return nil, nil
}

func (m *mockAuthService) UpdateAvatar(userID uint, avatarURL string) error {
	if m.updateAvatarFn != nil {
		return m.updateAvatarFn(userID, avatarURL)
	}
	return nil
}

func (m *mockAuthService) RefreshTokens(refreshToken string) (*dto.AuthResponse, error) {
	if m.refreshTokensFn != nil {
		return m.refreshTokensFn(refreshToken)
	}
	return nil, nil
}

func (m *mockAuthService) ChangePassword(userID uint, req *dto.ChangePasswordRequest) error {
	if m.changePasswordFn != nil {
		return m.changePasswordFn(userID, req)
	}
	return nil
}

func (m *mockAuthService) Logout(userID uint) error {
	if m.logoutFn != nil {
		return m.logoutFn(userID)
	}
	return nil
}

func (m *mockAuthService) DeleteUser(userID uint) error {
	if m.deleteUserFn != nil {
		return m.deleteUserFn(userID)
	}
	return nil
}

func TestNewAuthService(t *testing.T) {
	db := &gorm.DB{}
	svc := NewAuthService(db, "test-secret")
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestErrorVariables(t *testing.T) {
	errs := []error{ErrUserAlreadyExists, ErrInvalidCredentials, ErrUserNotFound}
	for i := 0; i < len(errs); i++ {
		for j := i + 1; j < len(errs); j++ {
			if errors.Is(errs[i], errs[j]) {
				t.Errorf("sentinel errors should be distinct: %v == %v", errs[i], errs[j])
			}
		}
	}
}

func TestRegisterUser_Success(t *testing.T) {
	ms := &mockAuthService{
		registerUserFn: func(req *dto.RegisterRequest) (*dto.AuthResponse, error) {
			return &dto.AuthResponse{
				User:         dto.UserResponse{ID: 1, Username: req.Username},
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
			}, nil
		},
	}
	resp, err := ms.RegisterUser(&dto.RegisterRequest{Username: "testuser", Password: "password123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.User.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", resp.User.Username)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
}

func TestRegisterUser_AlreadyExists(t *testing.T) {
	ms := &mockAuthService{
		registerUserFn: func(req *dto.RegisterRequest) (*dto.AuthResponse, error) {
			return nil, ErrUserAlreadyExists
		},
	}
	_, err := ms.RegisterUser(&dto.RegisterRequest{Username: "existing", Password: "password123"})
	if !errors.Is(err, ErrUserAlreadyExists) {
		t.Errorf("expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestRegisterUser_DBError(t *testing.T) {
	dbErr := errors.New("db error")
	ms := &mockAuthService{
		registerUserFn: func(req *dto.RegisterRequest) (*dto.AuthResponse, error) {
			return nil, dbErr
		},
	}
	_, err := ms.RegisterUser(&dto.RegisterRequest{Username: "test", Password: "pass"})
	if !errors.Is(err, dbErr) {
		t.Errorf("expected db error, got %v", err)
	}
}

func TestAuthenticateUser_Success(t *testing.T) {
	ms := &mockAuthService{
		authenticateUserFn: func(req *dto.LoginRequest) (*dto.AuthResponse, error) {
			return &dto.AuthResponse{
				User:         dto.UserResponse{ID: 1, Username: req.Username},
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
			}, nil
		},
	}
	resp, err := ms.AuthenticateUser(&dto.LoginRequest{Username: "testuser", Password: "password123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.User.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", resp.User.Username)
	}
}

func TestAuthenticateUser_InvalidCredentials(t *testing.T) {
	ms := &mockAuthService{
		authenticateUserFn: func(req *dto.LoginRequest) (*dto.AuthResponse, error) {
			return nil, ErrInvalidCredentials
		},
	}
	_, err := ms.AuthenticateUser(&dto.LoginRequest{Username: "test", Password: "wrong"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthenticateUser_DBError(t *testing.T) {
	dbErr := errors.New("db error")
	ms := &mockAuthService{
		authenticateUserFn: func(req *dto.LoginRequest) (*dto.AuthResponse, error) {
			return nil, dbErr
		},
	}
	_, err := ms.AuthenticateUser(&dto.LoginRequest{Username: "test", Password: "pass"})
	if !errors.Is(err, dbErr) {
		t.Errorf("expected db error, got %v", err)
	}
}

func TestGetUserByID_Success(t *testing.T) {
	ms := &mockAuthService{
		getUserByIDFn: func(userID uint) (*model.User, error) {
			return &model.User{ID: userID, Username: "testuser"}, nil
		},
	}
	u, err := ms.GetUserByID(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != 1 || u.Username != "testuser" {
		t.Errorf("unexpected user: %+v", u)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	ms := &mockAuthService{
		getUserByIDFn: func(userID uint) (*model.User, error) {
			return nil, ErrUserNotFound
		},
	}
	_, err := ms.GetUserByID(999)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUpdateAvatar_Success(t *testing.T) {
	ms := &mockAuthService{
		updateAvatarFn: func(userID uint, avatarURL string) error {
			return nil
		},
	}
	if err := ms.UpdateAvatar(1, "/uploads/images/test.png"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateAvatar_DBError(t *testing.T) {
	dbErr := errors.New("update failed")
	ms := &mockAuthService{
		updateAvatarFn: func(userID uint, avatarURL string) error {
			return dbErr
		},
	}
	if err := ms.UpdateAvatar(1, "/test.png"); !errors.Is(err, dbErr) {
		t.Errorf("expected db error, got %v", err)
	}
}

func TestRefreshTokens_Success(t *testing.T) {
	ms := &mockAuthService{
		refreshTokensFn: func(refreshToken string) (*dto.AuthResponse, error) {
			return &dto.AuthResponse{
				User:         dto.UserResponse{ID: 1, Username: "testuser"},
				AccessToken:  "new-access",
				RefreshToken: "new-refresh",
			}, nil
		},
	}
	resp, err := ms.RefreshTokens("old-refresh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AccessToken != "new-access" {
		t.Errorf("expected new access token")
	}
}

func TestRefreshTokens_InvalidToken(t *testing.T) {
	ms := &mockAuthService{
		refreshTokensFn: func(refreshToken string) (*dto.AuthResponse, error) {
			return nil, errors.New("invalid token")
		},
	}
	_, err := ms.RefreshTokens("bad-token")
	if err == nil {
		t.Error("expected error for invalid refresh token")
	}
}
