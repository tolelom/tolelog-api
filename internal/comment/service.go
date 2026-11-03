package comment

import (
	"errors"
	"tolelom_api/internal/model"

	"gorm.io/gorm"
)

var (
	ErrCommentNotFound = errors.New("comment not found")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrPostNotFound    = errors.New("post not found")
	ErrParentNotFound  = errors.New("parent comment not found")
)

type Service interface {
	CreateComment(comment *model.Comment) error
	GetCommentsByPostID(postID uint, viewerID *uint, limit int) ([]model.Comment, int64, error)
	UpdateComment(postID uint, commentID uint, userID uint, content string) (*model.Comment, error)
	DeleteComment(postID uint, commentID uint, userID uint) error
}

type service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

func (s *service) CreateComment(comment *model.Comment) error {
	if err := s.checkPostVisible(comment.PostID, &comment.UserID); err != nil {
		return err
	}
	if comment.ParentID != nil {
		if _, err := s.findPostComment(comment.PostID, *comment.ParentID); err != nil {
			if errors.Is(err, ErrCommentNotFound) {
				return ErrParentNotFound
			}
			return err
		}
	}

	if err := s.db.Create(comment).Error; err != nil {
		return err
	}
	return s.db.Preload("User").First(comment, comment.ID).Error
}

// checkPostVisible: 없는 글은 ErrPostNotFound, 보는 사람에게 숨은 글은 ErrUnauthorized.
func (s *service) checkPostVisible(postID uint, viewerID *uint) error {
	var post model.Post
	if err := s.db.Select("id, user_id, is_public, status").First(&post, postID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPostNotFound
		}
		return err
	}
	if !post.VisibleTo(viewerID) {
		return ErrUnauthorized
	}
	return nil
}

func (s *service) GetCommentsByPostID(postID uint, viewerID *uint, limit int) ([]model.Comment, int64, error) {
	if err := s.checkPostVisible(postID, viewerID); err != nil {
		return nil, 0, err
	}

	var comments []model.Comment
	var total int64

	query := s.db.Where("post_id = ?", postID)

	if err := query.Model(&model.Comment{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Preload("User").Order("created_at ASC").Limit(limit).Find(&comments).Error; err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

func (s *service) UpdateComment(postID uint, commentID uint, userID uint, content string) (*model.Comment, error) {
	comment, err := s.findPostComment(postID, commentID)
	if err != nil {
		return nil, err
	}

	if comment.UserID != userID {
		return nil, ErrUnauthorized
	}
	if err := s.checkPostVisible(postID, &userID); err != nil {
		return nil, err
	}

	// Save 는 고칠 행이 없으면(그 사이 삭제됨) INSERT … ON CONFLICT 로 행 전체를 다시 써서
	// 지운 댓글을 되살린다. 바꿀 칸만 쓴다.
	if err := s.db.Model(comment).Updates(map[string]any{"content": content, "is_edited": true}).Error; err != nil {
		return nil, err
	}

	if err := s.db.Preload("User").First(comment, comment.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}

	return comment, nil
}

// DeleteComment 는 글이 숨은 뒤에도 작성자가 자기 댓글을 지울 수 있게 글 가시성을 보지 않는다.
// 답글은 지우지 않는다(목록에서 고아 답글로 맨 위에 보인다).
func (s *service) DeleteComment(postID uint, commentID uint, userID uint) error {
	comment, err := s.findPostComment(postID, commentID)
	if err != nil {
		return err
	}

	if comment.UserID != userID {
		return ErrUnauthorized
	}

	return s.db.Delete(comment).Error
}

// findPostComment 는 다른 글의 댓글을 없는 댓글과 똑같이 다룬다.
func (s *service) findPostComment(postID uint, commentID uint) (*model.Comment, error) {
	var comment model.Comment
	if err := s.db.Where("id = ? AND post_id = ?", commentID, postID).First(&comment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}
	return &comment, nil
}
