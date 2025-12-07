package service

import (
	"errors"
	"tolelom_api/internal/config"
	"tolelom_api/internal/model"

	"gorm.io/gorm"
)

type PostService struct {
	db *gorm.DB
}

func NewPostService() *PostService {
	return &PostService{
		db: config.GetDB(),
	}
}

// CreatePost - 새 글 생성
func (ps *PostService) CreatePost(post *model.Post) error {
	if err := ps.db.Create(post).Error; err != nil {
		return err
	}
	return nil
}

// GetPostByID - ID로 글 조회
func (ps *PostService) GetPostByID(postID uint) (*model.Post, error) {
	var post model.Post
	if err := ps.db.Preload("User").First(&post, postID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("글을 찾을 수 없습니다")
		}
		return nil, err
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
func (ps *PostService) GetUserPosts(userID uint, page, pageSize int) ([]model.Post, int64, error) {
	var posts []model.Post
	var total int64

	// 사용자의 전체 글 개수
	if err := ps.db.Where("user_id = ?", userID).Model(&model.Post{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 페이지네이션 적용
	offset := (page - 1) * pageSize
	if err := ps.db.Preload("User").Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&posts).Error; err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

// UpdatePost - 글 수정
func (ps *PostService) UpdatePost(post *model.Post) error {
	if err := ps.db.Save(post).Error; err != nil {
		return err
	}
	return nil
}

// DeletePost - 글 삭제
func (ps *PostService) DeletePost(postID uint, userID uint) error {
	// 글이 사용자의 것인지 확인
	var post model.Post
	if err := ps.db.First(&post, postID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("글을 찾을 수 없습니다")
		}
		return err
	}

	if post.UserID != userID {
		return errors.New("이 글을 삭제할 권한이 없습니다")
	}

	if err := ps.db.Delete(&post).Error; err != nil {
		return err
	}
	return nil
}

// CheckOwnership - 글의 소유자인지 확인
func (ps *PostService) CheckOwnership(postID uint, userID uint) (bool, error) {
	var post model.Post
	if err := ps.db.First(&post, postID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, errors.New("글을 찾을 수 없습니다")
		}
		return false, err
	}

	return post.UserID == userID, nil
}
