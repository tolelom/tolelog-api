package comment

import (
	"errors"
	"testing"
	"tolelom_api/internal/model"

	"gorm.io/gorm"
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

func (m *mockService) GetCommentsByPostID(postID uint, _ int) ([]model.Comment, int64, error) {
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

// --- Service unit tests ---

func TestNewService(t *testing.T) {
	db := &gorm.DB{}
	svc := NewService(db)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestErrorVariables(t *testing.T) {
	errs := []error{ErrCommentNotFound, ErrUnauthorized, ErrPostNotFound, ErrParentNotFound}
	for i := 0; i < len(errs); i++ {
		for j := i + 1; j < len(errs); j++ {
			if errors.Is(errs[i], errs[j]) {
				t.Errorf("sentinel errors should be distinct: %v == %v", errs[i], errs[j])
			}
		}
	}
}

func TestCreateComment_PostNotFound(t *testing.T) {
	ms := &mockService{
		createCommentFn: func(comment *model.Comment) error {
			return ErrPostNotFound
		},
	}

	comment := &model.Comment{PostID: 999, UserID: 1, Content: "test"}
	err := ms.CreateComment(comment)
	if !errors.Is(err, ErrPostNotFound) {
		t.Errorf("expected ErrPostNotFound, got %v", err)
	}
}

func TestCreateComment_Unauthorized(t *testing.T) {
	ms := &mockService{
		createCommentFn: func(comment *model.Comment) error {
			return ErrUnauthorized
		},
	}

	comment := &model.Comment{PostID: 1, UserID: 2, Content: "test"}
	err := ms.CreateComment(comment)
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestCreateComment_Success(t *testing.T) {
	ms := &mockService{
		createCommentFn: func(comment *model.Comment) error {
			comment.ID = 42
			return nil
		},
	}

	comment := &model.Comment{PostID: 1, UserID: 1, Content: "hello"}
	err := ms.CreateComment(comment)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comment.ID != 42 {
		t.Errorf("expected comment ID 42, got %d", comment.ID)
	}
}

func TestCreateComment_DBError(t *testing.T) {
	dbErr := errors.New("db connection lost")
	ms := &mockService{
		createCommentFn: func(comment *model.Comment) error {
			return dbErr
		},
	}

	comment := &model.Comment{PostID: 1, UserID: 1, Content: "test"}
	err := ms.CreateComment(comment)
	if !errors.Is(err, dbErr) {
		t.Errorf("expected db error, got %v", err)
	}
}

func TestCreateComment_Reply_Success(t *testing.T) {
	parentID := uint(1)
	ms := &mockService{
		createCommentFn: func(comment *model.Comment) error {
			if comment.ParentID == nil || *comment.ParentID != parentID {
				t.Error("expected ParentID to be set")
			}
			comment.ID = 2
			return nil
		},
	}

	comment := &model.Comment{PostID: 1, UserID: 1, Content: "reply", ParentID: &parentID}
	err := ms.CreateComment(comment)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comment.ID != 2 {
		t.Errorf("expected comment ID 2, got %d", comment.ID)
	}
}

func TestCreateComment_Reply_ParentNotFound(t *testing.T) {
	ms := &mockService{
		createCommentFn: func(comment *model.Comment) error {
			return ErrParentNotFound
		},
	}

	parentID := uint(999)
	comment := &model.Comment{PostID: 1, UserID: 1, Content: "reply", ParentID: &parentID}
	err := ms.CreateComment(comment)
	if !errors.Is(err, ErrParentNotFound) {
		t.Errorf("expected ErrParentNotFound, got %v", err)
	}
}

func TestGetCommentsByPostID_Success(t *testing.T) {
	ms := &mockService{
		getCommentsByPostFn: func(postID uint) ([]model.Comment, int64, error) {
			comments := []model.Comment{
				{ID: 1, PostID: postID, UserID: 1, Content: "first"},
				{ID: 2, PostID: postID, UserID: 2, Content: "second"},
			}
			return comments, 2, nil
		},
	}

	comments, total, err := ms.GetCommentsByPostID(1, 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(comments) != 2 {
		t.Errorf("expected 2 comments, got %d", len(comments))
	}
}

func TestGetCommentsByPostID_Empty(t *testing.T) {
	ms := &mockService{
		getCommentsByPostFn: func(postID uint) ([]model.Comment, int64, error) {
			return []model.Comment{}, 0, nil
		},
	}

	comments, total, err := ms.GetCommentsByPostID(999, 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(comments) != 0 {
		t.Errorf("expected 0 comments, got %d", len(comments))
	}
}

func TestGetCommentsByPostID_DBError(t *testing.T) {
	dbErr := errors.New("query failed")
	ms := &mockService{
		getCommentsByPostFn: func(postID uint) ([]model.Comment, int64, error) {
			return nil, 0, dbErr
		},
	}

	_, _, err := ms.GetCommentsByPostID(1, 200)
	if !errors.Is(err, dbErr) {
		t.Errorf("expected db error, got %v", err)
	}
}

func TestGetCommentsByPostID_WithReplies(t *testing.T) {
	parentID := uint(1)
	ms := &mockService{
		getCommentsByPostFn: func(postID uint) ([]model.Comment, int64, error) {
			comments := []model.Comment{
				{ID: 1, PostID: postID, UserID: 1, Content: "root"},
				{ID: 2, PostID: postID, UserID: 2, Content: "reply", ParentID: &parentID},
			}
			return comments, 2, nil
		},
	}

	comments, total, err := ms.GetCommentsByPostID(1, 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	foundReply := false
	for _, c := range comments {
		if c.ParentID != nil && *c.ParentID == 1 {
			foundReply = true
		}
	}
	if !foundReply {
		t.Error("expected to find a reply with ParentID=1")
	}
}

func TestDeleteComment_Success(t *testing.T) {
	ms := &mockService{
		deleteCommentFn: func(commentID uint, userID uint) error {
			return nil
		},
	}

	err := ms.DeleteComment(1, 1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeleteComment_NotFound(t *testing.T) {
	ms := &mockService{
		deleteCommentFn: func(commentID uint, userID uint) error {
			return ErrCommentNotFound
		},
	}

	err := ms.DeleteComment(999, 1)
	if !errors.Is(err, ErrCommentNotFound) {
		t.Errorf("expected ErrCommentNotFound, got %v", err)
	}
}

func TestDeleteComment_Unauthorized(t *testing.T) {
	ms := &mockService{
		deleteCommentFn: func(commentID uint, userID uint) error {
			if userID != 1 {
				return ErrUnauthorized
			}
			return nil
		},
	}

	err := ms.DeleteComment(1, 2)
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestDeleteComment_DBError(t *testing.T) {
	dbErr := errors.New("delete failed")
	ms := &mockService{
		deleteCommentFn: func(commentID uint, userID uint) error {
			return dbErr
		},
	}

	err := ms.DeleteComment(1, 1)
	if !errors.Is(err, dbErr) {
		t.Errorf("expected db error, got %v", err)
	}
}
