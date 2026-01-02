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

func NewAuthService() *AuthService {
	return &AuthService{db: model.DB}
}

func (s *AuthService) RegisterUser(req *model.RegisterRequest) (*model.User, string, error) {
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))

	var existing model.User
	if err := s.db.Where("username = ?", req.Username).First(&existing).Error; err == nil {
		return nil, "", errors.New("user already exists")
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, "", err
	}

	now := time.Now()
	user := &model.User{
		Username:  req.Username,
		Password:  hash,
		CreatedAt: now,
		LastLogin: now,
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, "", err
	}

	token, err := utils.GenerateJWT(user)
	return user, token, err
}

func (s *AuthService) AuthenticateUser(req *model.LoginRequest) (*model.User, string, error) {
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))

	var user model.User
	if err := s.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", errors.New("invalid credentials")
		}
		return nil, "", err
	}
	if !utils.CheckPasswordHash(req.Password, user.Password) {
		return nil, "", errors.New("invalid credentials")
	}

	user.LastLogin = time.Now()
	if err := s.db.Save(&user).Error; err != nil {
		return nil, "", err
	}

	token, err := utils.GenerateJWT(&user)
	return &user, token, err
}
