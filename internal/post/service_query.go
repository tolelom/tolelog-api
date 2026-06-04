package post

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"tolelom_api/internal/model"
)

var validTagPattern = regexp.MustCompile(`^[\p{L}\p{N}\-_\.\+\# ]+$`)

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
