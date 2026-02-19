package post

import (
	"errors"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/model"

	"gorm.io/gorm"
)

var (
	ErrPostNotFound     = errors.New("post not found")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrNoFieldsToUpdate = errors.New("no fields to update")
)

type Service interface {
	CreatePost(post *model.Post) error
	GetPostByID(postID uint, userID *uint) (*model.Post, error)
	GetPublicPosts(page, pageSize int, tag string) ([]model.Post, int64, error)
	GetUserPosts(userID uint, currentUserID *uint, page, pageSize int, tag string) ([]model.Post, int64, error)
	UpdatePost(postID uint, userID uint, req *dto.UpdatePostRequest) (*model.Post, error)
	DeletePost(postID uint, userID uint) error
}

type service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

// CreatePost - 새 글 생성
func (s *service) CreatePost(post *model.Post) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(post).Error; err != nil {
			return err
		}
		// User 정보 로드
		if err := tx.Preload("User").First(post, post.ID).Error; err != nil {
			return err
		}
		return nil
	})
}

// GetPostByID - ID로 글 조회
func (s *service) GetPostByID(postID uint, userID *uint) (*model.Post, error) {
	var post model.Post
	if err := s.db.Preload("User").First(&post, postID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}

	if !post.IsPublic && (userID == nil || *userID != post.UserID) {
		return nil, ErrUnauthorized
	}

	return &post, nil
}

// GetPublicPosts - 공개 글 목록 조회 (페이지네이션, 태그 필터)
func (s *service) GetPublicPosts(page, pageSize int, tag string) ([]model.Post, int64, error) {
	var posts []model.Post
	var total int64

	query := s.db.Where("is_public = ?", true)
	if tag != "" {
		query = query.Where("tags LIKE ?", "%"+tag+"%")
	}

	if err := query.Model(&model.Post{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&posts).Error; err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

// GetUserPosts - 특정 사용자의 글 목록 조회 (페이지네이션, 태그 필터)
func (s *service) GetUserPosts(userID uint, currentUserID *uint, page, pageSize int, tag string) ([]model.Post, int64, error) {
	var posts []model.Post
	var total int64

	query := s.db.Model(&model.Post{}).Where("user_id = ?", userID)

	if currentUserID == nil || *currentUserID != userID {
		query = query.Where("is_public = ?", true)
	}
	if tag != "" {
		query = query.Where("tags LIKE ?", "%"+tag+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&posts).Error; err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

// UpdatePost - 글 수정
func (s *service) UpdatePost(postID uint, userID uint, req *dto.UpdatePostRequest) (*model.Post, error) {
	var post model.Post
	if err := s.db.First(&post, postID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}

	if post.UserID != userID {
		return nil, ErrUnauthorized
	}

	updates := map[string]interface{}{}

	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.IsPublic != nil {
		updates["is_public"] = *req.IsPublic
	}
	if req.Tags != nil {
		updates["tags"] = *req.Tags
	}

	if len(updates) == 0 {
		return nil, ErrNoFieldsToUpdate
	}

	if err := s.db.Model(&post).Updates(updates).Error; err != nil {
		return nil, err
	}

	if err := s.db.Preload("User").First(&post, postID).Error; err != nil {
		return nil, err
	}

	return &post, nil
}

// DeletePost - 글 삭제
func (s *service) DeletePost(postID uint, userID uint) error {
	var post model.Post
	if err := s.db.First(&post, postID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPostNotFound
		}
		return err
	}

	if post.UserID != userID {
		return ErrUnauthorized
	}

	if err := s.db.Delete(&post).Error; err != nil {
		return err
	}
	return nil
}
