package dto

import (
	"time"
	"tolelom_api/internal/model"
)

type CreateSeriesRequest struct {
	Title       string `json:"title" validate:"required,min=1,max=255"`
	Description string `json:"description,omitempty" validate:"max=2000"`
}

type UpdateSeriesRequest struct {
	Title       *string `json:"title,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=2000"`
}

type AddPostToSeriesRequest struct {
	PostID uint `json:"post_id" validate:"required"`
	Order  int  `json:"order,omitempty"`
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
	PrevPost     *SeriesNavPost `json:"prev_post" extensions:"x-nullable"`
	NextPost     *SeriesNavPost `json:"next_post" extensions:"x-nullable"`
}

type SeriesInfo struct {
	SeriesID    uint   `json:"series_id"`
	SeriesTitle string `json:"series_title"`
	SeriesOrder int    `json:"series_order"`
}

func SeriesToResponse(s *model.Series, postCount int) SeriesResponse {
	return SeriesResponse{
		ID:          s.ID,
		Title:       s.Title,
		Description: s.Description,
		UserID:      s.UserID,
		Author:      s.User.Username,
		PostCount:   postCount,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

func SeriesToDetailResponse(s *model.Series, posts []model.Post) SeriesDetailResponse {

	items := make([]SeriesPostItem, len(posts))
	for i := range posts {
		p := &posts[i]
		items[i] = SeriesPostItem{
			ID:        p.ID,
			Title:     p.Title,
			Order:     p.OrderInSeries(),
			CreatedAt: p.CreatedAt,
		}
	}

	return SeriesDetailResponse{
		ID:          s.ID,
		Title:       s.Title,
		Description: s.Description,
		UserID:      s.UserID,
		Author:      s.User.Username,
		Posts:       items,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}
