package dto

import (
	"time"
	"tolelom_api/internal/model"
)

type CreateCommentRequest struct {
	Content  string `json:"content" validate:"required,min=1,max=2000"`
	ParentID *uint  `json:"parent_id" validate:"omitempty"`
}

type UpdateCommentRequest struct {
	Content string `json:"content" validate:"required,min=1,max=2000"`
}

type CommentResponse struct {
	ID        uint              `json:"id"`
	PostID    uint              `json:"post_id"`
	UserID    uint              `json:"user_id"`
	ParentID  *uint             `json:"parent_id"`
	Author    string            `json:"author"`
	AvatarURL string            `json:"avatar_url"`
	Content   string            `json:"content"`
	IsEdited  bool              `json:"is_edited"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Replies   []CommentResponse `json:"replies"`
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
		ParentID:  c.ParentID,
		Author:    c.User.Username,
		AvatarURL: c.User.AvatarURL,
		Content:   c.Content,
		IsEdited:  c.IsEdited,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		Replies:   []CommentResponse{},
	}
}

// BuildCommentTree converts a flat list of comments into a tree structure.
// Root comments (ParentID == nil) are returned at the top level,
// with their replies nested under them.
func BuildCommentTree(comments []model.Comment) []CommentResponse {
	responseMap := make(map[uint]*CommentResponse, len(comments))

	// First pass: create all response objects
	for i := range comments {
		resp := CommentToResponse(&comments[i])
		responseMap[resp.ID] = &resp
	}

	// Second pass: attach children to parents
	var rootIDs []uint
	var orphanIDs []uint
	for i := range comments {
		if comments[i].ParentID != nil {
			if parent, ok := responseMap[*comments[i].ParentID]; ok {
				child := responseMap[comments[i].ID]
				parent.Replies = append(parent.Replies, *child)
			} else {
				orphanIDs = append(orphanIDs, comments[i].ID)
			}
		} else {
			rootIDs = append(rootIDs, comments[i].ID)
		}
	}

	// Third pass: collect roots (now with replies attached) and orphans
	roots := make([]CommentResponse, 0, len(rootIDs)+len(orphanIDs))
	for _, id := range rootIDs {
		roots = append(roots, *responseMap[id])
	}
	for _, id := range orphanIDs {
		roots = append(roots, *responseMap[id])
	}

	return roots
}
