package comment

import (
	"tolelom_api/internal/model"
)

// mockService implements Service for testing.
type mockService struct {
	createCommentFn     func(comment *model.Comment) error
	getCommentsByPostFn func(postID uint) ([]model.Comment, int64, error) // limit is ignored in mock
	updateCommentFn     func(commentID uint, userID uint, content string) (*model.Comment, error)
	deleteCommentFn     func(commentID uint, userID uint) error
}

func (m *mockService) CreateComment(comment *model.Comment) error {
	if m.createCommentFn != nil {
		return m.createCommentFn(comment)
	}
	return nil
}

func (m *mockService) GetCommentsByPostID(postID uint, _ *uint, _ int) ([]model.Comment, int64, error) {
	if m.getCommentsByPostFn != nil {
		return m.getCommentsByPostFn(postID)
	}
	return nil, 0, nil
}

func (m *mockService) UpdateComment(commentID uint, userID uint, content string) (*model.Comment, error) {
	if m.updateCommentFn != nil {
		return m.updateCommentFn(commentID, userID, content)
	}
	return nil, nil
}

func (m *mockService) DeleteComment(commentID uint, userID uint) error {
	if m.deleteCommentFn != nil {
		return m.deleteCommentFn(commentID, userID)
	}
	return nil
}
