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
	GetCommentsByPostID(postID uint, limit int) ([]model.Comment, int64, error)
	UpdateComment(commentID uint, userID uint, content string) (*model.Comment, error)
	DeleteComment(commentID uint, userID uint) error
}

type service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

// CreateComment verifies the post exists and is accessible, validates
// the parent comment if provided, then creates the comment.
func (s *service) CreateComment(comment *model.Comment) error {
	// Verify post exists and is not soft-deleted
	var post model.Post
	if err := s.db.Select("id, user_id, is_public").Where("id = ? AND deleted_at IS NULL", comment.PostID).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPostNotFound
		}
		return err
	}

	// Only allow comments on public posts or by the post author
	if !post.IsPublic && comment.UserID != post.UserID {
		return ErrUnauthorized
	}

	// Validate parent comment if this is a reply
	if comment.ParentID != nil {
		var parent model.Comment
		if err := s.db.Where("id = ? AND post_id = ?", *comment.ParentID, comment.PostID).First(&parent).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrParentNotFound
			}
			return err
		}
	}

	if err := s.db.Create(comment).Error; err != nil {
		return err
	}

	// Preload User for the response
	if err := s.db.Preload("User").First(comment, comment.ID).Error; err != nil {
		return err
	}

	return nil
}

// GetCommentsByPostID returns comments for a post ordered by creation time, up to the given limit.
func (s *service) GetCommentsByPostID(postID uint, limit int) ([]model.Comment, int64, error) {
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

// UpdateComment updates a comment's content if the requesting user is the author.
func (s *service) UpdateComment(commentID uint, userID uint, content string) (*model.Comment, error) {
	var comment model.Comment
	if err := s.db.First(&comment, commentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}

	if comment.UserID != userID {
		return nil, ErrUnauthorized
	}

	comment.Content = content
	comment.IsEdited = true
	if err := s.db.Save(&comment).Error; err != nil {
		return nil, err
	}

	// Preload User for the response
	if err := s.db.Preload("User").First(&comment, comment.ID).Error; err != nil {
		return nil, err
	}

	return &comment, nil
}

// DeleteComment soft-deletes a comment if the requesting user is the author.
// Child replies are preserved (they become orphans displayed as root-level).
func (s *service) DeleteComment(commentID uint, userID uint) error {
	var comment model.Comment
	if err := s.db.First(&comment, commentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCommentNotFound
		}
		return err
	}

	if comment.UserID != userID {
		return ErrUnauthorized
	}

	if err := s.db.Delete(&comment).Error; err != nil {
		return err
	}

	return nil
}
