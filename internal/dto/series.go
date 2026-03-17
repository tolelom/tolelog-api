package dto

import (
	"time"
	"tolelom_api/internal/model"
)

type CreateSeriesRequest struct {
	Title       string `json:"title" validate:"required,min=1,max=255"`
	Description string `json:"description" validate:"max=2000"`
}

type UpdateSeriesRequest struct {
	Title       *string `json:"title" validate:"omitempty,min=1,max=255"`
	Description *string `json:"description" validate:"omitempty,max=2000"`
}

type AddPostToSeriesRequest struct {
	PostID uint `json:"post_id" validate:"required"`
	Order  int  `json:"order"`
}

type ReorderPostsRequest struct {
	PostIDs []uint `json:"post_ids" validate:"required"`
}

type SeriesResponse struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	UserID      uint      `json:"user_id"`
	Author      string    `json:"author"`
	PostCount   int       `json:"post_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SeriesPostItem struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Order     int       `json:"order"`
	CreatedAt time.Time `json:"created_at"`
}

type SeriesDetailResponse struct {
	ID          uint             `json:"id"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	UserID      uint             `json:"user_id"`
	Author      string           `json:"author"`
	Posts       []SeriesPostItem `json:"posts"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

type SeriesNavPost struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
}

type SeriesNavResponse struct {
	SeriesID     uint           `json:"series_id"`
	SeriesTitle  string         `json:"series_title"`
	CurrentOrder int            `json:"current_order"`
	TotalPosts   int            `json:"total_posts"`
	PrevPost     *SeriesNavPost `json:"prev_post"`
	NextPost     *SeriesNavPost `json:"next_post"`
}

type SeriesInfo struct {
	SeriesID    uint   `json:"series_id"`
	SeriesTitle string `json:"series_title"`
	SeriesOrder int    `json:"series_order"`
}

func SeriesToResponse(s *model.Series, postCount int) SeriesResponse {
	author := ""
	if s.User.ID != 0 {
		author = s.User.Username
	}
	return SeriesResponse{
		ID:          s.ID,
		Title:       s.Title,
		Description: s.Description,
		UserID:      s.UserID,
		Author:      author,
		PostCount:   postCount,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

func SeriesToDetailResponse(s *model.Series, posts []model.Post) SeriesDetailResponse {
	author := ""
	if s.User.ID != 0 {
		author = s.User.Username
	}

	items := make([]SeriesPostItem, len(posts))
	for i, p := range posts {
		order := 0
		if p.SeriesOrder != nil {
			order = *p.SeriesOrder
		}
		items[i] = SeriesPostItem{
			ID:        p.ID,
			Title:     p.Title,
			Order:     order,
			CreatedAt: p.CreatedAt,
		}
	}

	return SeriesDetailResponse{
		ID:          s.ID,
		Title:       s.Title,
		Description: s.Description,
		UserID:      s.UserID,
		Author:      author,
		Posts:       items,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}
