package service

import (
	"errors"
	"tolelom_api/internal/model"

	"gorm.io/gorm"
)

type PostService struct {
	db *gorm.DB
}

func NewPostService(db *gorm.DB) *PostService {
	return &PostService{
		db: db,
	}
}

// CreatePost - 새 글 생성
func (ps *PostService) CreatePost(post *model.Post) (*model.Post, error) {
	if err := ps.db.Create(post).Error; err != nil {
		return nil, err
	}

	// User 정보 로드
	if err := ps.db.Preload("User").First(post, post.ID).Error; err != nil {
		return nil, err
	}

	return post, nil
}

// GetPostByID - ID로 글 조회
func (ps *PostService) GetPostByID(postID uint, userID *uint) (*model.Post, error) {
	var post model.Post
	if err := ps.db.Preload("User").First(&post, postID).Error; err != nil {
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

// GetPublicPosts - 공개 글 목록 조회 (페이지네이션)
func (ps *PostService) GetPublicPosts(page, pageSize int) ([]model.Post, int64, error) {
	var posts []model.Post
	var total int64

	// 전체 공개 글 개수
	if err := ps.db.Where("is_public = ?", true).Model(&model.Post{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 페이지네이션 적용
	offset := (page - 1) * pageSize
	if err := ps.db.Preload("User").Where("is_public = ?", true).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&posts).Error; err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

// GetUserPosts - 특정 사용자의 글 목록 조회 (페이지네이션)
func (ps *PostService) GetUserPosts(userID uint, currentUserID *uint, page, pageSize int) ([]model.Post, int64, error) {
	var posts []model.Post
	var total int64

	query := ps.db.Model(&model.Post{}).Where("user_id = ?", userID)

	if currentUserID == nil || *currentUserID != userID {
		query = query.Where("is_public = ?", true)
	}

	// 사용자의 전체 글 개수
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 페이지네이션 적용
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
func (ps *PostService) UpdatePost(postID uint, userID uint, req *model.UpdatePostRequest) (*model.Post, error) {
	// 권한 확인
	var post model.Post
	if err := ps.db.First(&post, postID).Error; err != nil {
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

	if len(updates) == 0 {
		return nil, ErrNoFieldsToUpdate
	}

	if err := ps.db.Model(&post).Updates(updates).Error; err != nil {
		return nil, err
	}

	if err := ps.db.Preload("User").First(&post, postID).Error; err != nil {
		return nil, err
	}

	return &post, nil
}

// DeletePost - 글 삭제
func (ps *PostService) DeletePost(postID uint, userID uint) error {
	// 글이 사용자의 것인지 확인
	var post model.Post
	if err := ps.db.First(&post, postID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPostNotFound
		}
		return err
	}

	if post.UserID != userID {
		return ErrUnauthorized
	}

	if err := ps.db.Delete(&post).Error; err != nil {
		return err
	}
	return nil
}
