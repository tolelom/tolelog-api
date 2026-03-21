package user

import (
	"errors"
	"strings"
	"time"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/model"
	"tolelom_api/internal/utils"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var (
	ErrUserAlreadyExists    = errors.New("user already exists")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidPassword      = errors.New("invalid current password")
	ErrSamePassword         = errors.New("new password is the same as current")
	ErrInvalidRefreshToken  = errors.New("invalid refresh token")
)

type AuthService interface {
	RegisterUser(req *dto.RegisterRequest) (*dto.AuthResponse, error)
	AuthenticateUser(req *dto.LoginRequest) (*dto.AuthResponse, error)
	GetUserByID(userID uint) (*model.User, error)
	UpdateAvatar(userID uint, avatarURL string) error
	RefreshTokens(refreshToken string) (*dto.AuthResponse, error)
	ChangePassword(userID uint, req *dto.ChangePasswordRequest) error
	Logout(userID uint) error
	DeleteUser(userID uint) error
}

type authService struct {
	db        *gorm.DB
	jwtSecret string
}

func NewAuthService(db *gorm.DB, jwtSecret string) AuthService {
	return &authService{db: db, jwtSecret: jwtSecret}
}

func (s *authService) RegisterUser(req *dto.RegisterRequest) (*dto.AuthResponse, error) {
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))

	var existing model.User
	if err := s.db.Where("username = ?", req.Username).First(&existing).Error; err == nil {
		return nil, ErrUserAlreadyExists
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	u := &model.User{
		Username:     req.Username,
		PasswordHash: hash,
		LastLogin:    time.Now(),
	}

	if err := s.db.Create(u).Error; err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}

	accessToken, refreshToken, err := utils.GenerateTokenPair(u, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		User:         dto.UserToResponse(u),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *authService) GetUserByID(userID uint) (*model.User, error) {
	var u model.User
	if err := s.db.First(&u, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (s *authService) AuthenticateUser(req *dto.LoginRequest) (*dto.AuthResponse, error) {
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))

	var u model.User
	if err := s.db.Where("username = ?", req.Username).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if !utils.CheckPasswordHash(req.Password, u.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	u.LastLogin = time.Now()
	if err := s.db.Save(&u).Error; err != nil {
		return nil, err
	}

	accessToken, refreshToken, err := utils.GenerateTokenPair(&u, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		User:         dto.UserToResponse(&u),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *authService) UpdateAvatar(userID uint, avatarURL string) error {
	return s.db.Model(&model.User{}).Where("id = ?", userID).Update("avatar_url", avatarURL).Error
}

func (s *authService) RefreshTokens(refreshToken string) (*dto.AuthResponse, error) {
	claims, err := utils.ValidateRefreshToken(refreshToken, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	u, err := s.GetUserByID(claims.UserID)
	if err != nil {
		return nil, err
	}

	if claims.TokenVersion != u.TokenVersion {
		return nil, ErrInvalidRefreshToken
	}

	accessToken, newRefreshToken, err := utils.GenerateTokenPair(u, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		User:         dto.UserToResponse(u),
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *authService) Logout(userID uint) error {
	return s.db.Model(&model.User{}).Where("id = ?", userID).UpdateColumn("token_version", gorm.Expr("token_version + 1")).Error
}

func (s *authService) DeleteUser(userID uint) error {
	if err := s.db.Delete(&model.User{}, userID).Error; err != nil {
		return err
	}
	return nil
}

func (s *authService) ChangePassword(userID uint, req *dto.ChangePasswordRequest) error {
	var u model.User
	if err := s.db.First(&u, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	if !utils.CheckPasswordHash(req.CurrentPassword, u.PasswordHash) {
		return ErrInvalidPassword
	}

	if utils.CheckPasswordHash(req.NewPassword, u.PasswordHash) {
		return ErrSamePassword
	}

	hash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	return s.db.Model(&u).Update("password", hash).Error
}
