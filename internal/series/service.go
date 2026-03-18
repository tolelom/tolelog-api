package series

import (
	"errors"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/model"

	"gorm.io/gorm"
)

var (
	ErrSeriesNotFound = errors.New("series not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrPostNotFound   = errors.New("post not found")
	ErrPostNotOwned   = errors.New("post not owned by user")
)

type Service interface {
	CreateSeries(series *model.Series) error
	GetSeriesByID(seriesID uint) (*model.Series, []model.Post, error)
	GetSeriesByUserID(userID uint) ([]model.Series, map[uint]int64, error)
	UpdateSeries(seriesID uint, userID uint, req *dto.UpdateSeriesRequest) (*model.Series, error)
	DeleteSeries(seriesID uint, userID uint) error
	AddPostToSeries(seriesID uint, postID uint, order int, userID uint) error
	RemovePostFromSeries(seriesID uint, postID uint, userID uint) error
	ReorderPosts(seriesID uint, postIDs []uint, userID uint) error
	GetSeriesNavigation(postID uint) (*dto.SeriesNavResponse, error)
	CountPostsInSeries(seriesID uint) (int64, error)
}

type service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

func (s *service) CreateSeries(series *model.Series) error {
	if err := s.db.Create(series).Error; err != nil {
		return err
	}
	return s.db.Preload("User").First(series, series.ID).Error
}

func (s *service) GetSeriesByID(seriesID uint) (*model.Series, []model.Post, error) {
	var series model.Series
	if err := s.db.Preload("User").First(&series, seriesID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrSeriesNotFound
		}
		return nil, nil, err
	}

	var posts []model.Post
	s.db.Where("series_id = ? AND deleted_at IS NULL", seriesID).
		Order("series_order ASC, created_at ASC").
		Find(&posts)

	return &series, posts, nil
}

func (s *service) GetSeriesByUserID(userID uint) ([]model.Series, map[uint]int64, error) {
	var seriesList []model.Series
	if err := s.db.Preload("User").Where("user_id = ?", userID).
		Order("created_at DESC").Find(&seriesList).Error; err != nil {
		return nil, nil, err
	}

	// Batch count posts per series in a single query
	postCounts := make(map[uint]int64, len(seriesList))
	if len(seriesList) > 0 {
		seriesIDs := make([]uint, len(seriesList))
		for i, ser := range seriesList {
			seriesIDs[i] = ser.ID
		}

		type countResult struct {
			SeriesID uint
			Count    int64
		}
		var results []countResult
		if err := s.db.Model(&model.Post{}).
			Select("series_id, COUNT(*) as count").
			Where("series_id IN ? AND deleted_at IS NULL", seriesIDs).
			Group("series_id").
			Find(&results).Error; err != nil {
			return nil, nil, err
		}
		for _, r := range results {
			postCounts[r.SeriesID] = r.Count
		}
	}

	return seriesList, postCounts, nil
}

func (s *service) UpdateSeries(seriesID uint, userID uint, req *dto.UpdateSeriesRequest) (*model.Series, error) {
	var series model.Series
	if err := s.db.First(&series, seriesID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSeriesNotFound
		}
		return nil, err
	}

	if series.UserID != userID {
		return nil, ErrUnauthorized
	}

	if req.Title != nil {
		series.Title = *req.Title
	}
	if req.Description != nil {
		series.Description = *req.Description
	}

	if err := s.db.Save(&series).Error; err != nil {
		return nil, err
	}

	s.db.Preload("User").First(&series, series.ID)
	return &series, nil
}

func (s *service) DeleteSeries(seriesID uint, userID uint) error {
	var series model.Series
	if err := s.db.First(&series, seriesID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSeriesNotFound
		}
		return err
	}

	if series.UserID != userID {
		return ErrUnauthorized
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 소속 글들의 series_id, series_order를 null로
		tx.Model(&model.Post{}).Where("series_id = ?", seriesID).
			Updates(map[string]interface{}{"series_id": nil, "series_order": nil})
		return tx.Delete(&series).Error
	})
}

func (s *service) AddPostToSeries(seriesID uint, postID uint, order int, userID uint) error {
	var series model.Series
	if err := s.db.First(&series, seriesID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSeriesNotFound
		}
		return err
	}
	if series.UserID != userID {
		return ErrUnauthorized
	}

	var post model.Post
	if err := s.db.Where("id = ? AND deleted_at IS NULL", postID).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPostNotFound
		}
		return err
	}
	if post.UserID != userID {
		return ErrPostNotOwned
	}

	post.SeriesID = &seriesID
	post.SeriesOrder = &order
	return s.db.Save(&post).Error
}

func (s *service) RemovePostFromSeries(seriesID uint, postID uint, userID uint) error {
	var series model.Series
	if err := s.db.First(&series, seriesID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSeriesNotFound
		}
		return err
	}
	if series.UserID != userID {
		return ErrUnauthorized
	}

	return s.db.Model(&model.Post{}).Where("id = ? AND series_id = ?", postID, seriesID).
		Updates(map[string]interface{}{"series_id": nil, "series_order": nil}).Error
}

func (s *service) ReorderPosts(seriesID uint, postIDs []uint, userID uint) error {
	var series model.Series
	if err := s.db.First(&series, seriesID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSeriesNotFound
		}
		return err
	}
	if series.UserID != userID {
		return ErrUnauthorized
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		for i, postID := range postIDs {
			order := i + 1
			if err := tx.Model(&model.Post{}).Where("id = ? AND series_id = ?", postID, seriesID).
				Update("series_order", order).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *service) CountPostsInSeries(seriesID uint) (int64, error) {
	var count int64
	if err := s.db.Model(&model.Post{}).
		Where("series_id = ? AND deleted_at IS NULL", seriesID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (s *service) GetSeriesNavigation(postID uint) (*dto.SeriesNavResponse, error) {
	var post model.Post
	if err := s.db.Where("id = ? AND deleted_at IS NULL", postID).First(&post).Error; err != nil {
		return nil, nil // 글이 없으면 nil 반환 (에러 아님)
	}

	if post.SeriesID == nil {
		return nil, nil // 시리즈에 속하지 않으면 nil
	}

	var series model.Series
	if err := s.db.First(&series, *post.SeriesID).Error; err != nil {
		return nil, nil
	}

	var posts []model.Post
	s.db.Select("id, title, series_order").
		Where("series_id = ? AND deleted_at IS NULL", *post.SeriesID).
		Order("series_order ASC, created_at ASC").
		Find(&posts)

	currentOrder := 0
	if post.SeriesOrder != nil {
		currentOrder = *post.SeriesOrder
	}

	nav := &dto.SeriesNavResponse{
		SeriesID:     series.ID,
		SeriesTitle:  series.Title,
		CurrentOrder: currentOrder,
		TotalPosts:   len(posts),
	}

	for i, p := range posts {
		if p.ID == postID {
			if i > 0 {
				nav.PrevPost = &dto.SeriesNavPost{ID: posts[i-1].ID, Title: posts[i-1].Title}
			}
			if i < len(posts)-1 {
				nav.NextPost = &dto.SeriesNavPost{ID: posts[i+1].ID, Title: posts[i+1].Title}
			}
			break
		}
	}

	return nav, nil
}
