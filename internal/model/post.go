package model

import "time"

type Post struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	Title     string    `gorm:"size:255;not null" validate:"required,min=1,max=255"`
	Content   string    `gorm:"type:longtext;not null" validate:"required"`
	UserID    uint      `gorm:"not null;index" validate:"required"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	IsPublic  bool      `gorm:"default:true"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type CreatePostRequest struct {
	Title    string `json:"title" binding:"required,min=1,max=255"`
	Content  string `json:"content" binding:"required"`
	IsPublic bool   `json:"is_public"`
}

type UpdatePostRequest struct {
	Title    string `json:"title" binding:"required,min=1,max=255"`
	Content  string `json:"content" binding:"required"`
	IsPublic bool   `json:"is_public"`
}

type PostResponse struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	UserID    uint      `json:"user_id"`
	Author    string    `json:"author"`
	IsPublic  bool      `json:"is_public"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PostListResponse struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	UserID    uint      `json:"user_id"`
	Author    string    `json:"author"`
	IsPublic  bool      `json:"is_public"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (p *Post) ToResponse() PostResponse {
	return PostResponse{
		ID:        p.ID,
		Title:     p.Title,
		Content:   p.Content,
		UserID:    p.UserID,
		Author:    p.User.Username,
		IsPublic:  p.IsPublic,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func (p *Post) ToListResponse() PostListResponse {
	return PostListResponse{
		ID:        p.ID,
		Title:     p.Title,
		UserID:    p.UserID,
		Author:    p.User.Username,
		IsPublic:  p.IsPublic,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
