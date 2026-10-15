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
	GetSeriesByID(seriesID uint, viewerID *uint) (*model.Series, []model.Post, error)
	GetSeriesByUserID(userID uint, viewerID *uint) ([]model.Series, map[uint]int64, error)
	UpdateSeries(seriesID uint, userID uint, req *dto.UpdateSeriesRequest) (*model.Series, error)
	DeleteSeries(seriesID uint, userID uint) error
	AddPostToSeries(seriesID uint, postID uint, order int, userID uint) error
	RemovePostFromSeries(seriesID uint, postID uint, userID uint) error
	ReorderPosts(seriesID uint, postIDs []uint, userID uint) error
	GetSeriesNavigation(postID uint, viewerID *uint) (*dto.SeriesNavResponse, error)
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

// seriesPostsVisibleTo scopes a series' posts to what the viewer may see:
// the series owner sees every post, others only public published ones.
func (s *service) seriesPostsVisibleTo(series *model.Series, viewerID *uint) *gorm.DB {
	query := s.db.Where("series_id = ? AND deleted_at IS NULL", series.ID)
	if viewerID == nil || *viewerID != series.UserID {
		query = query.Scopes(model.PublicPosts)
	}
	return query
}

func (s *service) GetSeriesByID(seriesID uint, viewerID *uint) (*model.Series, []model.Post, error) {
	var series model.Series
	if err := s.db.Preload("User").First(&series, seriesID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrSeriesNotFound
		}
		return nil, nil, err
	}

	var posts []model.Post
	s.seriesPostsVisibleTo(&series, viewerID).
		Order("series_order ASC, created_at ASC").
		Find(&posts)

	return &series, posts, nil
}

// GetSeriesByUserID 의 글 수는 보는 사람이 볼 수 있는 글만 센다(작성자는 전부).
func (s *service) GetSeriesByUserID(userID uint, viewerID *uint) ([]model.Series, map[uint]int64, error) {
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
		query := s.db.Model(&model.Post{}).
			Select("series_id, COUNT(*) as count").
			Where("series_id IN ?", seriesIDs)
		if viewerID == nil || *viewerID != userID {
			query = query.Scopes(model.PublicPosts)
		}
		if err := query.Group("series_id").Find(&results).Error; err != nil {
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

// AddPostToSeries 는 order 가 0 이하면 순서를 정한다. 이미 같은 시리즈에 있는 글은 지금 자리를 지키고,
// 새로 들어오는 글은 맨 뒤에 붙는다(에디터는 저장할 때마다 order 0 으로 부른다).
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

	if order <= 0 {
		if post.SeriesID != nil && *post.SeriesID == seriesID && post.SeriesOrder != nil {
			order = *post.SeriesOrder
		} else {
			var maxOrder int
			if err := s.db.Model(&model.Post{}).
				Where("series_id = ? AND deleted_at IS NULL", seriesID).
				Select("COALESCE(MAX(series_order), 0)").Scan(&maxOrder).Error; err != nil {
				return err
			}
			order = maxOrder + 1
		}
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

func (s *service) GetSeriesNavigation(postID uint, viewerID *uint) (*dto.SeriesNavResponse, error) {
	var post model.Post
	if err := s.db.Where("id = ? AND deleted_at IS NULL", postID).First(&post).Error; err != nil {
		return nil, nil // 글이 없으면 nil 반환 (에러 아님)
	}
	if !post.VisibleTo(viewerID) {
		return nil, nil // 볼 수 없는 글은 시리즈 소속도 드러내지 않는다
	}

	if post.SeriesID == nil {
		return nil, nil // 시리즈에 속하지 않으면 nil
	}

	var series model.Series
	if err := s.db.First(&series, *post.SeriesID).Error; err != nil {
		return nil, nil
	}

	var posts []model.Post
	s.seriesPostsVisibleTo(&series, viewerID).
		Select("id, title, series_order").
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
