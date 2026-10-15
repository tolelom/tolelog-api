package user

import (
	"tolelom_api/internal/dto"
	"tolelom_api/internal/model"
)

// mockAuthService implements AuthService for testing.
type mockAuthService struct {
	registerUserFn     func(req *dto.RegisterRequest) (*dto.AuthResponse, error)
	authenticateUserFn func(req *dto.LoginRequest) (*dto.AuthResponse, error)
	getUserByIDFn      func(userID uint) (*model.User, error)
	updateAvatarFn     func(userID uint, avatarURL string) error
	refreshTokensFn    func(refreshToken string) (*dto.AuthResponse, error)
	changePasswordFn   func(userID uint, req *dto.ChangePasswordRequest) (*dto.AuthResponse, error)
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

func (m *mockAuthService) ChangePassword(userID uint, req *dto.ChangePasswordRequest) (*dto.AuthResponse, error) {
	if m.changePasswordFn != nil {
		return m.changePasswordFn(userID, req)
	}
	return newAuthResponse(), nil
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
