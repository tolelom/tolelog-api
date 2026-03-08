package dto

import (
	"time"
	"tolelom_api/internal/model"
)

type LoginRequest struct {
	Username string `json:"username" validate:"required,min=4,max=20,alphanum_underscore"`
	Password string `json:"password" validate:"required,min=6,max=128"`
}

type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=4,max=20,alphanum_underscore"`
	Password string `json:"password" validate:"required,min=6,max=128"`
}

type AuthResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type UserResponse struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
	LastLogin time.Time `json:"last_login"`
}

func UserToResponse(u *model.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		AvatarURL: u.AvatarURL,
		CreatedAt: u.CreatedAt,
		LastLogin: u.LastLogin,
	}
}
