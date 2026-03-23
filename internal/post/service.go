package post

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"tolelom_api/internal/cache"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/model"

	"gorm.io/gorm"
)

var validTagPattern = regexp.MustCompile(`^[\p{L}\p{N}\-_\.\+\# ]+$`)

var (
	ErrPostNotFound       = errors.New("post not found")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrNoFieldsToUpdate   = errors.New("no fields to update")
	ErrInvalidTag         = errors.New("invalid tag parameter")
	ErrInvalidSearchQuery = errors.New("invalid search query")
)

// Cache key patterns and TTLs
const (
	cachePublicPosts = "posts:public:%d:%d:%s" // page, pageSize, tag
	cachePost        = "posts:%d"              // postID
	cacheTTLList     = 2 * time.Minute
	cacheTTLPost     = 5 * time.Minute
)

// validSearchPattern allows Unicode letters, numbers, spaces, and common punctuation.
// Disallows SQL LIKE wildcards (%, _) and special characters that could cause issues.
var validSearchPattern = regexp.MustCompile(`^[\p{L}\p{N}\s\-_\.\,\!\?\:\;\'\"\(\)]+$`)

// SanitizeSearchQuery validates and sanitizes a search query parameter.
func SanitizeSearchQuery(query string) (string, error) {
	query = strings.TrimSpace(query)
	if len(query) < 2 || len(query) > 100 {
		return "", ErrInvalidSearchQuery
	}
	if !validSearchPattern.MatchString(query) {
		return "", ErrInvalidSearchQuery
	}
	return query, nil
}

// SanitizeTag validates and sanitizes a tag parameter.
func SanitizeTag(tag string) (string, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return "", nil
	}
	if len(tag) > 50 {
		return "", ErrInvalidTag
	}
	if !validTagPattern.MatchString(tag) {
		return "", ErrInvalidTag
	}
	return tag, nil
}

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

// invalidatePostCaches invalidates list caches and optionally a specific post cache.
func (s *service) invalidatePostCaches(postID uint) {
	if s.cache == nil {
		return
	}
	if err := s.cache.DeleteByPattern("posts:public:*"); err != nil {
		slog.Warn("캐시 삭제 실패 (목록)", "error", err)
	}
	if postID > 0 {
		if err := s.cache.Delete(fmt.Sprintf(cachePost, postID)); err != nil {
			slog.Warn("캐시 삭제 실패 (개별 글)", "postID", postID, "error", err)
		}
	}
}

// cachedPublicPostList is the cached structure for public post list responses.
type cachedPublicPostList struct {
	Posts []model.Post `json:"posts"`
	Total int64        `json:"total"`
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

// GetPublicPosts - 공개 글 목록 조회 (페이지네이션, 태그 필터)
func (s *service) GetPublicPosts(page, pageSize int, tag string) ([]model.Post, int64, error) {
	// Try cache
	if s.cache != nil {
		cacheKey := fmt.Sprintf(cachePublicPosts, page, pageSize, tag)
		var cached cachedPublicPostList
		if err := s.cache.Get(cacheKey, &cached); err == nil {
			return cached.Posts, cached.Total, nil
		}
	}

	var posts []model.Post
	var total int64

	query := s.db.Where("is_public = ? AND status = ?", true, "published")
	if tag != "" {
		sanitized, err := SanitizeTag(tag)
		if err != nil {
			return nil, 0, err
		}
		if sanitized != "" {
			query = query.Where("id IN (?)",
				s.db.Table("post_tags").
					Select("post_tags.post_id").
					Joins("JOIN tags ON tags.id = post_tags.tag_id").
					Where("tags.name = ?", sanitized))
		}
	}

	if err := query.Model(&model.Post{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Preload("User").Preload("Tags").Preload("Series").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&posts).Error; err != nil {
		return nil, 0, err
	}

	// Store in cache
	if s.cache != nil {
		cacheKey := fmt.Sprintf(cachePublicPosts, page, pageSize, tag)
		if err := s.cache.Set(cacheKey, &cachedPublicPostList{Posts: posts, Total: total}, cacheTTLList); err != nil {
			slog.Warn("캐시 저장 실패 (목록)", "error", err)
		}
	}

	return posts, total, nil
}

// GetUserPosts - 특정 사용자의 글 목록 조회 (페이지네이션, 태그 필터)
func (s *service) GetUserPosts(userID uint, currentUserID *uint, page, pageSize int, tag string) ([]model.Post, int64, error) {
	var posts []model.Post
	var total int64

	query := s.db.Model(&model.Post{}).Where("user_id = ?", userID)

	if currentUserID == nil || *currentUserID != userID {
		query = query.Where("is_public = ? AND status = ?", true, "published")
	}
	if tag != "" {
		sanitized, err := SanitizeTag(tag)
		if err != nil {
			return nil, 0, err
		}
		if sanitized != "" {
			query = query.Where("id IN (?)",
				s.db.Table("post_tags").
					Select("post_tags.post_id").
					Joins("JOIN tags ON tags.id = post_tags.tag_id").
					Where("tags.name = ?", sanitized))
		}
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Preload("User").Preload("Tags").Preload("Series").
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

// escapeLike escapes SQL LIKE metacharacters (%, _) in a string.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// SearchPosts - 공개 글 검색 (제목/본문 LIKE, 페이지네이션)
func (s *service) SearchPosts(query string, page, pageSize int) ([]model.Post, int64, error) {
	var posts []model.Post
	var total int64

	likeQuery := "%" + escapeLike(query) + "%"
	q := s.db.Where("is_public = ? AND status = ? AND (title LIKE ? OR content LIKE ?)", true, "published", likeQuery, likeQuery)

	if err := q.Model(&model.Post{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := q.Preload("User").Preload("Tags").Preload("Series").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&posts).Error; err != nil {
		return nil, 0, err
	}

	return posts, total, nil
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

// ToggleLike toggles a like for a post. Returns the new liked state and total like count.
func (s *service) ToggleLike(postID uint, userID uint) (bool, uint, error) {
	var liked bool
	var likeCount uint

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var existing model.PostLike
		findErr := tx.Where("post_id = ? AND user_id = ?", postID, userID).First(&existing).Error

		if findErr == nil {
			// Already liked → unlike
			if err := tx.Delete(&existing).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.Post{}).Where("id = ? AND like_count > 0", postID).
				UpdateColumn("like_count", gorm.Expr("like_count - 1")).Error; err != nil {
				return err
			}
			liked = false
		} else if errors.Is(findErr, gorm.ErrRecordNotFound) {
			// Not liked → like
			like := model.PostLike{PostID: postID, UserID: userID}
			if err := tx.Create(&like).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.Post{}).Where("id = ?", postID).
				UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error; err != nil {
				return err
			}
			liked = true
		} else {
			return findErr
		}

		// Get updated count within transaction
		var post model.Post
		if err := tx.Select("like_count").First(&post, postID).Error; err != nil {
			return err
		}
		likeCount = post.LikeCount
		return nil
	})
	if err != nil {
		return false, 0, err
	}

	s.invalidatePostCaches(postID)

	return liked, likeCount, nil
}

// GetDrafts returns all draft posts for a user.
func (s *service) GetDrafts(userID uint) ([]model.Post, error) {
	var posts []model.Post
	if err := s.db.Where("user_id = ? AND status = ?", userID, "draft").
		Preload("User").Preload("Tags").
		Order("updated_at DESC").
		Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}

// IsLiked checks if a user has liked a post.
func (s *service) IsLiked(postID uint, userID uint) bool {
	var count int64
	s.db.Model(&model.PostLike{}).Where("post_id = ? AND user_id = ?", postID, userID).Count(&count)
	return count > 0
}
