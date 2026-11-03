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

func (s *service) findOwnedSeries(seriesID, userID uint) (*model.Series, error) {
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
	return &series, nil
}

func (s *service) CreateSeries(series *model.Series) error {
	if err := s.db.Create(series).Error; err != nil {
		return err
	}
	return s.db.Preload("User").First(series, series.ID).Error
}

func isOwner(viewerID *uint, ownerID uint) bool {
	return viewerID != nil && *viewerID == ownerID
}

// visibleTo hides non-public posts from everyone but the series owner.
func visibleTo(query *gorm.DB, ownerID uint, viewerID *uint) *gorm.DB {
	if isOwner(viewerID, ownerID) {
		return query
	}
	return query.Scopes(model.PublicPosts)
}

func (s *service) seriesPostsVisibleTo(series *model.Series, viewerID *uint) *gorm.DB {
	return visibleTo(s.db.Model(&model.Post{}).Where("series_id = ?", series.ID), series.UserID, viewerID)
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
	if err := s.seriesPostsVisibleTo(&series, viewerID).
		Order("series_order ASC, created_at ASC").
		Find(&posts).Error; err != nil {
		return nil, nil, err
	}

	return &series, posts, nil
}

func (s *service) GetSeriesByUserID(userID uint, viewerID *uint) ([]model.Series, map[uint]int64, error) {
	var seriesList []model.Series
	if err := s.db.Preload("User").Where("user_id = ?", userID).
		Order("created_at DESC").Find(&seriesList).Error; err != nil {
		return nil, nil, err
	}
	postCounts, err := s.visiblePostCounts(seriesList, userID, viewerID)
	if err != nil {
		return nil, nil, err
	}
	return seriesList, postCounts, nil
}

func (s *service) visiblePostCounts(seriesList []model.Series, ownerID uint, viewerID *uint) (map[uint]int64, error) {
	counts := make(map[uint]int64, len(seriesList))
	if len(seriesList) == 0 {
		return counts, nil
	}
	seriesIDs := make([]uint, len(seriesList))
	for i := range seriesList {
		seriesIDs[i] = seriesList[i].ID
	}

	var rows []struct {
		SeriesID uint
		Count    int64
	}
	query := s.db.Model(&model.Post{}).
		Select("series_id, COUNT(*) as count").
		Where("series_id IN ?", seriesIDs)
	if err := visibleTo(query, ownerID, viewerID).Group("series_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		counts[r.SeriesID] = r.Count
	}
	return counts, nil
}

func (s *service) UpdateSeries(seriesID uint, userID uint, req *dto.UpdateSeriesRequest) (*model.Series, error) {
	series, err := s.findOwnedSeries(seriesID, userID)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	// Save 는 고칠 행이 없으면(그 사이 삭제됨) INSERT … ON CONFLICT 로 행 전체를 다시 써서
	// 지운 시리즈를 되살린다. 바꿀 칸만 쓴다.
	if len(updates) > 0 {
		if err := s.db.Model(series).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	if err := s.db.Preload("User").First(series, series.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSeriesNotFound
		}
		return nil, err
	}
	return series, nil
}

func (s *service) DeleteSeries(seriesID uint, userID uint) error {
	series, err := s.findOwnedSeries(seriesID, userID)
	if err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Post{}).Where("series_id = ?", seriesID).Updates(outOfSeries()).Error; err != nil {
			return err
		}
		return tx.Delete(series).Error
	})
}

// AddPostToSeries 는 order 가 0 이하면 순서를 정한다. 이미 같은 시리즈에 있는 글은 지금 자리를 지키고,
// 새로 들어오는 글은 맨 뒤에 붙는다(에디터는 저장할 때마다 order 0 으로 부른다).
func (s *service) AddPostToSeries(seriesID uint, postID uint, order int, userID uint) error {
	if _, err := s.findOwnedSeries(seriesID, userID); err != nil {
		return err
	}

	var post model.Post
	if err := s.db.First(&post, postID).Error; err != nil {
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
				Where("series_id = ?", seriesID).
				Select("COALESCE(MAX(series_order), 0)").Scan(&maxOrder).Error; err != nil {
				return err
			}
			order = maxOrder + 1
		}
	}

	// 시리즈 칸만 쓴다. Save 는 글 전체를 다시 써서 그 사이 다른 요청이 고친
	// 제목·본문을 덮을 수 있다.
	return s.db.Model(&post).Updates(map[string]any{
		"series_id":    seriesID,
		"series_order": order,
	}).Error
}

func (s *service) RemovePostFromSeries(seriesID uint, postID uint, userID uint) error {
	if _, err := s.findOwnedSeries(seriesID, userID); err != nil {
		return err
	}

	return s.db.Model(&model.Post{}).Where("id = ? AND series_id = ?", postID, seriesID).Updates(outOfSeries()).Error
}

func outOfSeries() map[string]any {
	return map[string]any{"series_id": nil, "series_order": nil}
}

func (s *service) ReorderPosts(seriesID uint, postIDs []uint, userID uint) error {
	if _, err := s.findOwnedSeries(seriesID, userID); err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		for i, postID := range postIDs {
			if err := tx.Model(&model.Post{}).Where("id = ? AND series_id = ?", postID, seriesID).
				Update("series_order", i+1).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *service) CountPostsInSeries(seriesID uint) (int64, error) {
	var count int64
	if err := s.db.Model(&model.Post{}).
		Where("series_id = ?", seriesID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// GetSeriesNavigation 은 글이 없거나, 보는 사람에게 숨었거나, 시리즈에 없으면 오류 없이 nil 이다.
// 숨은 글은 시리즈 소속도 드러내지 않는다.
func (s *service) GetSeriesNavigation(postID uint, viewerID *uint) (*dto.SeriesNavResponse, error) {
	var post model.Post
	if err := s.db.First(&post, postID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if !post.VisibleTo(viewerID) || post.SeriesID == nil {
		return nil, nil
	}

	var series model.Series
	if err := s.db.First(&series, *post.SeriesID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	var posts []model.Post
	if err := s.seriesPostsVisibleTo(&series, viewerID).
		Select("id, title, series_order").
		Order("series_order ASC, created_at ASC").
		Find(&posts).Error; err != nil {
		return nil, err
	}

	nav := &dto.SeriesNavResponse{
		SeriesID:     series.ID,
		SeriesTitle:  series.Title,
		CurrentOrder: post.OrderInSeries(),
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
