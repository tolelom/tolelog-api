package dto

import (
	"strings"
	"time"
	"tolelom_api/internal/model"
)

type CreatePostRequest struct {
	Title    string `json:"title" validate:"required,min=1,max=255"`
	Content  string `json:"content" validate:"required,min=1,max=500000"`
	IsPublic bool   `json:"is_public"`
	Tags     string `json:"tags" validate:"max=500"`
}

type UpdatePostRequest struct {
	Title    *string `json:"title" validate:"omitempty,min=1,max=255"`
	Content  *string `json:"content" validate:"omitempty,min=1,max=500000"`
	IsPublic *bool   `json:"is_public"`
	Tags     *string `json:"tags" validate:"omitempty,max=500"`
}

type PostResponse struct {
	ID        uint        `json:"id"`
	Title     string      `json:"title"`
	Content   string      `json:"content"`
	UserID    uint        `json:"user_id"`
	Author    string      `json:"author"`
	IsPublic  bool        `json:"is_public"`
	Tags      string      `json:"tags"`
	Series    *SeriesInfo `json:"series,omitempty"`
	ViewCount uint        `json:"view_count"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type PostListResponse struct {
	ID        uint        `json:"id"`
	Title     string      `json:"title"`
	UserID    uint        `json:"user_id"`
	Author    string      `json:"author"`
	IsPublic  bool        `json:"is_public"`
	Tags      string      `json:"tags"`
	Series    *SeriesInfo `json:"series,omitempty"`
	ViewCount uint        `json:"view_count"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type PostListWithPagination struct {
	Posts      []PostListResponse `json:"posts"`
	Pagination Pagination        `json:"pagination"`
}

type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

func tagsToString(tags []model.Tag) string {
	names := make([]string, len(tags))
	for i, t := range tags {
		names[i] = t.Name
	}
	return strings.Join(names, ",")
}

func postSeriesInfo(p *model.Post) *SeriesInfo {
	if p.SeriesID != nil && p.Series != nil {
		order := 0
		if p.SeriesOrder != nil {
			order = *p.SeriesOrder
		}
		return &SeriesInfo{
			SeriesID:    *p.SeriesID,
			SeriesTitle: p.Series.Title,
			SeriesOrder: order,
		}
	}
	return nil
}

func PostToResponse(p *model.Post) PostResponse {
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
		Tags:      tagsToString(p.Tags),
		Series:    postSeriesInfo(p),
		ViewCount: p.ViewCount,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func PostToListResponse(p *model.Post) PostListResponse {
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
		Tags:      tagsToString(p.Tags),
		Series:    postSeriesInfo(p),
		ViewCount: p.ViewCount,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
