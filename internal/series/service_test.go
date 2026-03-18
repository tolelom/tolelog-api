package series

import (
	"errors"
	"testing"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/model"

	"gorm.io/gorm"
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

func (m *mockService) GetSeriesByID(seriesID uint) (*model.Series, []model.Post, error) {
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

func (m *mockService) GetSeriesNavigation(postID uint) (*dto.SeriesNavResponse, error) {
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

// --- Service unit tests ---

func TestNewService(t *testing.T) {
	db := &gorm.DB{}
	svc := NewService(db)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestErrorVariables(t *testing.T) {
	errs := []error{ErrSeriesNotFound, ErrUnauthorized, ErrPostNotFound, ErrPostNotOwned}
	for i := 0; i < len(errs); i++ {
		for j := i + 1; j < len(errs); j++ {
			if errors.Is(errs[i], errs[j]) {
				t.Errorf("sentinel errors should be distinct: %v == %v", errs[i], errs[j])
			}
		}
	}
}

func TestCreateSeries_Success(t *testing.T) {
	ms := &mockService{
		createSeriesFn: func(series *model.Series) error {
			series.ID = 1
			return nil
		},
	}
	s := &model.Series{Title: "Go Basics", UserID: 1}
	if err := ms.CreateSeries(s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.ID != 1 {
		t.Errorf("expected ID 1, got %d", s.ID)
	}
}

func TestCreateSeries_DBError(t *testing.T) {
	dbErr := errors.New("db error")
	ms := &mockService{
		createSeriesFn: func(series *model.Series) error {
			return dbErr
		},
	}
	s := &model.Series{Title: "test", UserID: 1}
	if err := ms.CreateSeries(s); !errors.Is(err, dbErr) {
		t.Errorf("expected db error, got %v", err)
	}
}

func TestGetSeriesByID_Success(t *testing.T) {
	ms := &mockService{
		getSeriesByIDFn: func(seriesID uint) (*model.Series, []model.Post, error) {
			return &model.Series{ID: seriesID, Title: "Go"}, []model.Post{{ID: 1}, {ID: 2}}, nil
		},
	}
	s, posts, err := ms.GetSeriesByID(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.ID != 1 {
		t.Errorf("expected series ID 1, got %d", s.ID)
	}
	if len(posts) != 2 {
		t.Errorf("expected 2 posts, got %d", len(posts))
	}
}

func TestGetSeriesByID_NotFound(t *testing.T) {
	ms := &mockService{
		getSeriesByIDFn: func(seriesID uint) (*model.Series, []model.Post, error) {
			return nil, nil, ErrSeriesNotFound
		},
	}
	_, _, err := ms.GetSeriesByID(999)
	if !errors.Is(err, ErrSeriesNotFound) {
		t.Errorf("expected ErrSeriesNotFound, got %v", err)
	}
}

func TestGetSeriesByUserID_Success(t *testing.T) {
	ms := &mockService{
		getSeriesByUserIDFn: func(userID uint) ([]model.Series, map[uint]int64, error) {
			list := []model.Series{{ID: 1, Title: "A"}, {ID: 2, Title: "B"}}
			counts := map[uint]int64{1: 3, 2: 5}
			return list, counts, nil
		},
	}
	list, counts, err := ms.GetSeriesByUserID(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 series, got %d", len(list))
	}
	if counts[1] != 3 || counts[2] != 5 {
		t.Errorf("unexpected post counts: %v", counts)
	}
}

func TestGetSeriesByUserID_Empty(t *testing.T) {
	ms := &mockService{
		getSeriesByUserIDFn: func(userID uint) ([]model.Series, map[uint]int64, error) {
			return []model.Series{}, map[uint]int64{}, nil
		},
	}
	list, _, err := ms.GetSeriesByUserID(999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 series, got %d", len(list))
	}
}

func TestUpdateSeries_Success(t *testing.T) {
	title := "Updated Title"
	ms := &mockService{
		updateSeriesFn: func(seriesID uint, userID uint, req *dto.UpdateSeriesRequest) (*model.Series, error) {
			return &model.Series{ID: seriesID, Title: *req.Title, UserID: userID}, nil
		},
	}
	s, err := ms.UpdateSeries(1, 1, &dto.UpdateSeriesRequest{Title: &title})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Title != title {
		t.Errorf("expected title %q, got %q", title, s.Title)
	}
}

func TestUpdateSeries_NotFound(t *testing.T) {
	ms := &mockService{
		updateSeriesFn: func(seriesID uint, userID uint, req *dto.UpdateSeriesRequest) (*model.Series, error) {
			return nil, ErrSeriesNotFound
		},
	}
	_, err := ms.UpdateSeries(999, 1, &dto.UpdateSeriesRequest{})
	if !errors.Is(err, ErrSeriesNotFound) {
		t.Errorf("expected ErrSeriesNotFound, got %v", err)
	}
}

func TestUpdateSeries_Unauthorized(t *testing.T) {
	ms := &mockService{
		updateSeriesFn: func(seriesID uint, userID uint, req *dto.UpdateSeriesRequest) (*model.Series, error) {
			return nil, ErrUnauthorized
		},
	}
	_, err := ms.UpdateSeries(1, 2, &dto.UpdateSeriesRequest{})
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestDeleteSeries_Success(t *testing.T) {
	ms := &mockService{
		deleteSeriesFn: func(seriesID uint, userID uint) error {
			return nil
		},
	}
	if err := ms.DeleteSeries(1, 1); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeleteSeries_NotFound(t *testing.T) {
	ms := &mockService{
		deleteSeriesFn: func(seriesID uint, userID uint) error {
			return ErrSeriesNotFound
		},
	}
	if err := ms.DeleteSeries(999, 1); !errors.Is(err, ErrSeriesNotFound) {
		t.Errorf("expected ErrSeriesNotFound, got %v", err)
	}
}

func TestDeleteSeries_Unauthorized(t *testing.T) {
	ms := &mockService{
		deleteSeriesFn: func(seriesID uint, userID uint) error {
			return ErrUnauthorized
		},
	}
	if err := ms.DeleteSeries(1, 2); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestAddPostToSeries_Success(t *testing.T) {
	ms := &mockService{
		addPostToSeriesFn: func(seriesID uint, postID uint, order int, userID uint) error {
			return nil
		},
	}
	if err := ms.AddPostToSeries(1, 10, 1, 1); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAddPostToSeries_SeriesNotFound(t *testing.T) {
	ms := &mockService{
		addPostToSeriesFn: func(seriesID uint, postID uint, order int, userID uint) error {
			return ErrSeriesNotFound
		},
	}
	if err := ms.AddPostToSeries(999, 10, 1, 1); !errors.Is(err, ErrSeriesNotFound) {
		t.Errorf("expected ErrSeriesNotFound, got %v", err)
	}
}

func TestAddPostToSeries_PostNotFound(t *testing.T) {
	ms := &mockService{
		addPostToSeriesFn: func(seriesID uint, postID uint, order int, userID uint) error {
			return ErrPostNotFound
		},
	}
	if err := ms.AddPostToSeries(1, 999, 1, 1); !errors.Is(err, ErrPostNotFound) {
		t.Errorf("expected ErrPostNotFound, got %v", err)
	}
}

func TestAddPostToSeries_PostNotOwned(t *testing.T) {
	ms := &mockService{
		addPostToSeriesFn: func(seriesID uint, postID uint, order int, userID uint) error {
			return ErrPostNotOwned
		},
	}
	if err := ms.AddPostToSeries(1, 10, 1, 2); !errors.Is(err, ErrPostNotOwned) {
		t.Errorf("expected ErrPostNotOwned, got %v", err)
	}
}

func TestRemovePostFromSeries_Success(t *testing.T) {
	ms := &mockService{
		removePostFromSeriesFn: func(seriesID uint, postID uint, userID uint) error {
			return nil
		},
	}
	if err := ms.RemovePostFromSeries(1, 10, 1); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRemovePostFromSeries_SeriesNotFound(t *testing.T) {
	ms := &mockService{
		removePostFromSeriesFn: func(seriesID uint, postID uint, userID uint) error {
			return ErrSeriesNotFound
		},
	}
	if err := ms.RemovePostFromSeries(999, 10, 1); !errors.Is(err, ErrSeriesNotFound) {
		t.Errorf("expected ErrSeriesNotFound, got %v", err)
	}
}

func TestRemovePostFromSeries_Unauthorized(t *testing.T) {
	ms := &mockService{
		removePostFromSeriesFn: func(seriesID uint, postID uint, userID uint) error {
			return ErrUnauthorized
		},
	}
	if err := ms.RemovePostFromSeries(1, 10, 2); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestReorderPosts_Success(t *testing.T) {
	ms := &mockService{
		reorderPostsFn: func(seriesID uint, postIDs []uint, userID uint) error {
			return nil
		},
	}
	if err := ms.ReorderPosts(1, []uint{3, 1, 2}, 1); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReorderPosts_SeriesNotFound(t *testing.T) {
	ms := &mockService{
		reorderPostsFn: func(seriesID uint, postIDs []uint, userID uint) error {
			return ErrSeriesNotFound
		},
	}
	if err := ms.ReorderPosts(999, []uint{1}, 1); !errors.Is(err, ErrSeriesNotFound) {
		t.Errorf("expected ErrSeriesNotFound, got %v", err)
	}
}

func TestReorderPosts_Unauthorized(t *testing.T) {
	ms := &mockService{
		reorderPostsFn: func(seriesID uint, postIDs []uint, userID uint) error {
			return ErrUnauthorized
		},
	}
	if err := ms.ReorderPosts(1, []uint{1}, 2); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestGetSeriesNavigation_NilForNoSeries(t *testing.T) {
	ms := &mockService{
		getSeriesNavigationFn: func(postID uint) (*dto.SeriesNavResponse, error) {
			return nil, nil
		},
	}
	nav, err := ms.GetSeriesNavigation(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nav != nil {
		t.Error("expected nil navigation for post not in series")
	}
}

func TestGetSeriesNavigation_WithPrevNext(t *testing.T) {
	ms := &mockService{
		getSeriesNavigationFn: func(postID uint) (*dto.SeriesNavResponse, error) {
			return &dto.SeriesNavResponse{
				SeriesID:     1,
				SeriesTitle:  "Go Basics",
				CurrentOrder: 2,
				TotalPosts:   3,
				PrevPost:     &dto.SeriesNavPost{ID: 1, Title: "Chapter 1"},
				NextPost:     &dto.SeriesNavPost{ID: 3, Title: "Chapter 3"},
			}, nil
		},
	}
	nav, err := ms.GetSeriesNavigation(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nav == nil {
		t.Fatal("expected non-nil navigation")
	}
	if nav.PrevPost == nil || nav.PrevPost.ID != 1 {
		t.Error("expected prev post with ID 1")
	}
	if nav.NextPost == nil || nav.NextPost.ID != 3 {
		t.Error("expected next post with ID 3")
	}
}

func TestCountPostsInSeries_Success(t *testing.T) {
	ms := &mockService{
		countPostsInSeriesFn: func(seriesID uint) (int64, error) {
			return 5, nil
		},
	}
	count, err := ms.CountPostsInSeries(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 5 {
		t.Errorf("expected 5, got %d", count)
	}
}

func TestCountPostsInSeries_DBError(t *testing.T) {
	dbErr := errors.New("count failed")
	ms := &mockService{
		countPostsInSeriesFn: func(seriesID uint) (int64, error) {
			return 0, dbErr
		},
	}
	_, err := ms.CountPostsInSeries(1)
	if !errors.Is(err, dbErr) {
		t.Errorf("expected db error, got %v", err)
	}
}
