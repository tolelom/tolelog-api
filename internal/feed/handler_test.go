package feed

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/model"
	"tolelom_api/internal/post"

	"github.com/gofiber/fiber/v2"
)

// mockPostService implements post.Service for testing.
type mockPostService struct {
	getPublicPostsFn func(page, pageSize int, tag string) ([]model.Post, int64, error)
}

func (m *mockPostService) CreatePost(p *model.Post) error { return nil }
func (m *mockPostService) GetPostByID(postID uint, userID *uint) (*model.Post, error) {
	return nil, nil
}
func (m *mockPostService) GetPublicPosts(page, pageSize int, tag string) ([]model.Post, int64, error) {
	if m.getPublicPostsFn != nil {
		return m.getPublicPostsFn(page, pageSize, tag)
	}
	return nil, 0, nil
}
func (m *mockPostService) GetUserPosts(userID uint, currentUserID *uint, page, pageSize int, tag string) ([]model.Post, int64, error) {
	return nil, 0, nil
}
func (m *mockPostService) UpdatePost(postID uint, userID uint, req *dto.UpdatePostRequest) (*model.Post, error) {
	return nil, nil
}
func (m *mockPostService) DeletePost(postID uint, userID uint) error { return nil }
func (m *mockPostService) SearchPosts(query string, page, pageSize int) ([]model.Post, int64, error) {
	return nil, 0, nil
}
func (m *mockPostService) ToggleLike(postID uint, userID uint) (bool, uint, error) {
	return false, 0, nil
}
func (m *mockPostService) IsLiked(postID uint, userID uint) bool  { return false }
func (m *mockPostService) GetDrafts(userID uint) ([]model.Post, error) { return nil, nil }

var _ post.Service = (*mockPostService)(nil)

func setupFeedApp(svc post.Service) *fiber.App {
	app := fiber.New()
	h := NewHandler(svc)
	app.Get("/feed", h.Feed)
	return app
}

func TestFeed_Success(t *testing.T) {
	now := time.Now()
	svc := &mockPostService{
		getPublicPostsFn: func(page, pageSize int, tag string) ([]model.Post, int64, error) {
			return []model.Post{
				{Title: "테스트 글", Content: "내용", CreatedAt: now},
			}, 1, nil
		},
	}
	app := setupFeedApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/feed", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		t.Error("missing Content-Type header")
	}
}

func TestFeed_Empty(t *testing.T) {
	svc := &mockPostService{
		getPublicPostsFn: func(page, pageSize int, tag string) ([]model.Post, int64, error) {
			return nil, 0, nil
		},
	}
	app := setupFeedApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/feed", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestFeed_ServiceError(t *testing.T) {
	svc := &mockPostService{
		getPublicPostsFn: func(page, pageSize int, tag string) ([]model.Post, int64, error) {
			return nil, 0, errors.New("db error")
		},
	}
	app := setupFeedApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/feed", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}
}

func TestExcerpt(t *testing.T) {
	short := "짧은 글"
	result := excerpt(short, 200)
	if result != short {
		t.Errorf("excerpt short text = %q, want %q", result, short)
	}

	long := ""
	for i := 0; i < 250; i++ {
		long += "가"
	}
	result = excerpt(long, 200)
	runes := []rune(result)
	// 200 chars + "..." (3 chars)
	if len(runes) != 203 {
		t.Errorf("excerpt long text rune length = %d, want 203", len(runes))
	}
}

func TestUintToStr(t *testing.T) {
	tests := []struct {
		input    uint
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{12345, "12345"},
	}
	for _, tt := range tests {
		result := uintToStr(tt.input)
		if result != tt.expected {
			t.Errorf("uintToStr(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
