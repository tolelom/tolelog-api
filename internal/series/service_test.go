package series

import (
	"tolelom_api/internal/dto"
	"tolelom_api/internal/model"
)

// mockService implements Service for testing.
type mockService struct {
	createSeriesFn         func(series *model.Series) error
	getSeriesByIDFn        func(seriesID uint) (*model.Series, []model.Post, error)
	getSeriesByUserIDFn    func(userID uint) ([]model.Series, map[uint]int64, error)
	updateSeriesFn         func(seriesID uint, userID uint, req *dto.UpdateSeriesRequest) (*model.Series, error)
	deleteSeriesFn         func(seriesID uint, userID uint) error
	addPostToSeriesFn      func(seriesID uint, postID uint, order int, userID uint) error
	removePostFromSeriesFn func(seriesID uint, postID uint, userID uint) error
	reorderPostsFn         func(seriesID uint, postIDs []uint, userID uint) error
	getSeriesNavigationFn  func(postID uint) (*dto.SeriesNavResponse, error)
	countPostsInSeriesFn   func(seriesID uint) (int64, error)
}

func (m *mockService) CreateSeries(series *model.Series) error {
	if m.createSeriesFn != nil {
		return m.createSeriesFn(series)
	}
	return nil
}

func (m *mockService) GetSeriesByID(seriesID uint, _ *uint) (*model.Series, []model.Post, error) {
	if m.getSeriesByIDFn != nil {
		return m.getSeriesByIDFn(seriesID)
	}
	return nil, nil, nil
}

func (m *mockService) GetSeriesByUserID(userID uint) ([]model.Series, map[uint]int64, error) {
	if m.getSeriesByUserIDFn != nil {
		return m.getSeriesByUserIDFn(userID)
	}
	return nil, nil, nil
}

func (m *mockService) UpdateSeries(seriesID uint, userID uint, req *dto.UpdateSeriesRequest) (*model.Series, error) {
	if m.updateSeriesFn != nil {
		return m.updateSeriesFn(seriesID, userID, req)
	}
	return nil, nil
}

func (m *mockService) DeleteSeries(seriesID uint, userID uint) error {
	if m.deleteSeriesFn != nil {
		return m.deleteSeriesFn(seriesID, userID)
	}
	return nil
}

func (m *mockService) AddPostToSeries(seriesID uint, postID uint, order int, userID uint) error {
	if m.addPostToSeriesFn != nil {
		return m.addPostToSeriesFn(seriesID, postID, order, userID)
	}
	return nil
}

func (m *mockService) RemovePostFromSeries(seriesID uint, postID uint, userID uint) error {
	if m.removePostFromSeriesFn != nil {
		return m.removePostFromSeriesFn(seriesID, postID, userID)
	}
	return nil
}

func (m *mockService) ReorderPosts(seriesID uint, postIDs []uint, userID uint) error {
	if m.reorderPostsFn != nil {
		return m.reorderPostsFn(seriesID, postIDs, userID)
	}
	return nil
}

func (m *mockService) GetSeriesNavigation(postID uint, _ *uint) (*dto.SeriesNavResponse, error) {
	if m.getSeriesNavigationFn != nil {
		return m.getSeriesNavigationFn(postID)
	}
	return nil, nil
}

func (m *mockService) CountPostsInSeries(seriesID uint) (int64, error) {
	if m.countPostsInSeriesFn != nil {
		return m.countPostsInSeriesFn(seriesID)
	}
	return 0, nil
}
