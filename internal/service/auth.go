package service

import (
	"errors"
	"strings"
	"time"
	"tolelom_api/internal/model"
	"tolelom_api/internal/utils"

	"gorm.io/gorm"
)

type AuthService struct {
	db *gorm.DB
}

func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{db: db}
}

func (s *AuthService) RegisterUser(req *model.RegisterRequest) (*model.AuthResponse, error) {
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))

	var existing model.User
	if err := s.db.Where("username = ?", req.Username).First(&existing).Error; err == nil {
		return nil, ErrUserAlreadyExists
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:     req.Username,
		PasswordHash: hash,
		LastLogin:    time.Now(),
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	token, err := utils.GenerateJWT(user)
	return &model.AuthResponse{
		User:  user.ToResponse(),
		Token: token,
	}, err
}

func (s *AuthService) AuthenticateUser(req *model.LoginRequest) (*model.AuthResponse, error) {
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))

	var user model.User
	if err := s.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	user.LastLogin = time.Now()
	if err := s.db.Save(&user).Error; err != nil {
		return nil, err
	}

	token, err := utils.GenerateJWT(&user)
	return &model.AuthResponse{
		User:  user.ToResponse(),
		Token: token,
	}, err
}
