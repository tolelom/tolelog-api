package dto

import (
	"time"
	"tolelom_api/internal/model"
)

type CreateCommentRequest struct {
	Content string `json:"content" validate:"required,min=1,max=2000"`
}

type CommentResponse struct {
	ID        uint      `json:"id"`
	PostID    uint      `json:"post_id"`
	UserID    uint      `json:"user_id"`
	Author    string    `json:"author"`
	AvatarURL string    `json:"avatar_url"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type CommentListResponse struct {
	Comments []CommentResponse `json:"comments"`
	Total    int64             `json:"total"`
}

func CommentToResponse(c *model.Comment) CommentResponse {
	return CommentResponse{
		ID:        c.ID,
		PostID:    c.PostID,
		UserID:    c.UserID,
		Author:    c.User.Username,
		AvatarURL: c.User.AvatarURL,
		Content:   c.Content,
		CreatedAt: c.CreatedAt,
	}
}
