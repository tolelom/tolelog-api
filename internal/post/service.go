package post

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"tolelom_api/internal/cache"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/model"

	"gorm.io/gorm"
)

var (
	ErrPostNotFound       = errors.New("post not found")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrNoFieldsToUpdate   = errors.New("no fields to update")
	ErrInvalidTag         = errors.New("invalid tag parameter")
	ErrInvalidSearchQuery = errors.New("invalid search query")
)

type Service interface {
	CreatePost(post *model.Post) error
	GetPostByID(postID uint, userID *uint) (*model.Post, error)
	GetPublicPosts(page, pageSize int, tag string) ([]model.Post, int64, error)
	GetUserPosts(userID uint, currentUserID *uint, page, pageSize int, tag string) ([]model.Post, int64, error)
	UpdatePost(postID uint, userID uint, req *dto.UpdatePostRequest) (*model.Post, error)
	DeletePost(postID uint, userID uint) error
	SearchPosts(query string, page, pageSize int) ([]model.Post, int64, error)
	ToggleLike(postID uint, userID uint) (liked bool, likeCount uint, err error)
	IsLiked(postID uint, userID uint) bool
	GetDrafts(userID uint) ([]model.Post, error)
}

type service struct {
	db    *gorm.DB
	cache *cache.Cache
}

// NewService creates a new post service. cache can be nil for graceful degradation.
func NewService(db *gorm.DB, cache *cache.Cache) Service {
	return &service{db: db, cache: cache}
}

func splitTags(tagsStr string) []string {
	var result []string
	for _, t := range strings.Split(tagsStr, ",") {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func (s *service) syncTags(db *gorm.DB, post *model.Post, tagsStr string) error {
	tagNames := splitTags(tagsStr)
	if len(tagNames) == 0 {
		return db.Model(post).Association("Tags").Clear()
	}
	var tags []model.Tag
	for _, name := range tagNames {
		var tag model.Tag
		if err := db.Where("name = ?", name).FirstOrCreate(&tag, model.Tag{Name: name}).Error; err != nil {
			return err
		}
		tags = append(tags, tag)
	}
	return db.Model(post).Association("Tags").Replace(tags)
}

// CreatePost - 새 글 생성
func (s *service) CreatePost(post *model.Post) error {
	if post.Status == "" {
		post.Status = "published"
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(post).Error; err != nil {
			return err
		}
		// Sync tags from TagsRaw
		if post.TagsRaw != "" {
			if err := s.syncTags(tx, post, post.TagsRaw); err != nil {
				return err
			}
		}
		// Reload with User and Tags
		if err := tx.Preload("User").Preload("Tags").First(post, post.ID).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.invalidatePostCaches(0)
	return nil
}

// GetPostByID - ID로 글 조회
func (s *service) GetPostByID(postID uint, userID *uint) (*model.Post, error) {
	// Try cache for public posts (when no specific user context needed for cache key)
	if s.cache != nil {
		var post model.Post
		cacheKey := fmt.Sprintf(cachePost, postID)
		if err := s.cache.Get(cacheKey, &post); err == nil {
			// Cache hit — still enforce access control
			if !post.IsPublic && (userID == nil || *userID != post.UserID) {
				return nil, ErrUnauthorized
			}
			// Increment view count in DB (non-blocking for cached reads)
			go func() {
				_ = s.db.Model(&model.Post{}).Where("id = ?", postID).UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
			}()
			post.ViewCount++
			// Update cache with incremented view count
			if err := s.cache.Set(cacheKey, &post, cacheTTLPost); err != nil {
				slog.Warn("캐시 갱신 실패 (조회수)", "postID", postID, "error", err)
			}
			return &post, nil
		}
	}

	var post model.Post
	if err := s.db.Preload("User").Preload("Tags").Preload("Series").First(&post, postID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}

	if !post.IsPublic && (userID == nil || *userID != post.UserID) {
		return nil, ErrUnauthorized
	}

	// Increment view count
	_ = s.db.Model(&model.Post{}).Where("id = ?", postID).UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
	post.ViewCount++

	// Cache the post (regardless of public/private — access control is checked on retrieval)
	if s.cache != nil {
		cacheKey := fmt.Sprintf(cachePost, postID)
		if err := s.cache.Set(cacheKey, &post, cacheTTLPost); err != nil {
			slog.Warn("캐시 저장 실패 (개별 글)", "postID", postID, "error", err)
		}
	}

	return &post, nil
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
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if len(updates) == 0 {
		return nil, ErrNoFieldsToUpdate
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&post).Updates(updates).Error; err != nil {
			return err
		}

		// Sync tags if tags were updated
		if req.Tags != nil {
			if err := s.syncTags(tx, &post, *req.Tags); err != nil {
				return err
			}
		}

		// Reload with User and Tags
		if err := tx.Preload("User").Preload("Tags").First(&post, postID).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	s.invalidatePostCaches(postID)

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

	s.invalidatePostCaches(postID)

	return nil
}
