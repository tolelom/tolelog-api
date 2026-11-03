package dto

import (
	"time"
	"tolelom_api/internal/model"
)

type CreateCommentRequest struct {
	Content  string `json:"content" validate:"required,min=1,max=2000"`
	ParentID *uint  `json:"parent_id,omitempty" validate:"omitempty"`
}

type UpdateCommentRequest struct {
	Content string `json:"content" validate:"required,min=1,max=2000"`
}

type CommentResponse struct {
	ID        uint              `json:"id"`
	PostID    uint              `json:"post_id"`
	UserID    uint              `json:"user_id"`
	ParentID  *uint             `json:"parent_id" extensions:"x-nullable"`
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

// BuildCommentTree nests each comment under its parent. Replies whose parent is
// missing from the list (deleted, or past the fetch limit) are shown at the top
// level after the root comments.
func BuildCommentTree(comments []model.Comment) []CommentResponse {
	listed := make(map[uint]bool, len(comments))
	for i := range comments {
		listed[comments[i].ID] = true
	}

	var roots, orphans []*model.Comment
	repliesOf := make(map[uint][]*model.Comment)
	for i := range comments {
		c := &comments[i]
		switch {
		case c.ParentID == nil:
			roots = append(roots, c)
		case listed[*c.ParentID]:
			repliesOf[*c.ParentID] = append(repliesOf[*c.ParentID], c)
		default:
			orphans = append(orphans, c)
		}
	}

	var withReplies func(c *model.Comment) CommentResponse
	withReplies = func(c *model.Comment) CommentResponse {
		resp := CommentToResponse(c)
		for _, reply := range repliesOf[c.ID] {
			resp.Replies = append(resp.Replies, withReplies(reply))
		}
		return resp
	}

	tree := make([]CommentResponse, 0, len(roots)+len(orphans))
	for _, c := range append(roots, orphans...) {
		tree = append(tree, withReplies(c))
	}
	return tree
}
