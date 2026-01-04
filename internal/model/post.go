package model

import "time"

type Post struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	Title     string    `gorm:"size:255;not null"`
	Content   string    `gorm:"type:longtext;not null"`
	UserID    uint      `gorm:"not null;index"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	IsPublic  bool      `gorm:"default:true"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type CreatePostRequest struct {
	Title    string `json:"title" binding:"required,min=1,max=255"`
	Content  string `json:"content" binding:"required"`
	IsPublic *bool  `json:"is_public"`
}

type UpdatePostRequest struct {
	Title    *string `json:"title" binding:"omitempty,min=1,max=255"`
	Content  *string `json:"content" binding:"omitempty,min=1"`
	IsPublic *bool   `json:"is_public"`
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
	author := ""
	if p.User.ID != 0 {
		author = p.User.Username
	}

	return PostResponse{
		ID:        p.ID,
		Title:     p.Title,
		Content:   p.Content,
		UserID:    p.UserID,
		Author:    author,
		IsPublic:  p.IsPublic,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func (p *Post) ToListResponse() PostListResponse {
	author := ""
	if p.User.ID != 0 {
		author = p.User.Username
	}

	return PostListResponse{
		ID:        p.ID,
		Title:     p.Title,
		UserID:    p.UserID,
		Author:    author,
		IsPublic:  p.IsPublic,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
