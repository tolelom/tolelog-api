package post

import (
	"regexp"
	"strings"
	"tolelom_api/internal/model"
	"unicode/utf8"

	"gorm.io/gorm"
)

const (
	searchQueryMinRunes = 2
	searchQueryMaxRunes = 100
)

var validTagPattern = regexp.MustCompile(`^[\p{L}\p{N}\-_\.\+\# ]+$`)

// validSearchPattern allows Unicode letters, numbers, spaces, and common punctuation.
// Disallows SQL LIKE wildcards (%, _) and special characters that could cause issues.
var validSearchPattern = regexp.MustCompile(`^[\p{L}\p{N}\s\-_\.\,\!\?\:\;\'\"\(\)]+$`)

// SanitizeSearchQuery validates and sanitizes a search query parameter.
func SanitizeSearchQuery(query string) (string, error) {
	query = strings.TrimSpace(query)
	if n := utf8.RuneCountInString(query); n < searchQueryMinRunes || n > searchQueryMaxRunes {
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
	if utf8.RuneCountInString(tag) > model.TagNameMaxRunes {
		return "", ErrInvalidTag
	}
	if !validTagPattern.MatchString(tag) {
		return "", ErrInvalidTag
	}
	return tag, nil
}

func (s *service) GetPublicPosts(page, pageSize int, tag string) ([]model.Post, int64, error) {
	query, err := s.taggedWith(s.db.Model(&model.Post{}).Scopes(model.PublicPosts), tag)
	if err != nil {
		return nil, 0, err
	}
	return newestFirstPage(query, page, pageSize)
}

func (s *service) GetUserPosts(userID uint, currentUserID *uint, page, pageSize int, tag string) ([]model.Post, int64, error) {
	query := s.db.Model(&model.Post{}).Where("user_id = ?", userID)
	if currentUserID == nil || *currentUserID != userID {
		query = query.Scopes(model.PublicPosts)
	}
	query, err := s.taggedWith(query, tag)
	if err != nil {
		return nil, 0, err
	}
	return newestFirstPage(query, page, pageSize)
}

// taggedWith narrows query to posts carrying tag. An empty tag leaves it unchanged.
func (s *service) taggedWith(query *gorm.DB, tag string) (*gorm.DB, error) {
	sanitized, err := SanitizeTag(tag)
	if err != nil || sanitized == "" {
		return query, err
	}
	return query.Where("id IN (?)",
		s.db.Table("post_tags").
			Select("post_tags.post_id").
			Joins("JOIN tags ON tags.id = post_tags.tag_id").
			Where("tags.name = ?", sanitized)), nil
}

// newestFirstPage returns one page of query's posts with their details, and the total match count.
func newestFirstPage(query *gorm.DB, page, pageSize int) ([]model.Post, int64, error) {
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var posts []model.Post
	if err := query.Scopes(withDetails).
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
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

func (s *service) SearchPosts(query string, page, pageSize int) ([]model.Post, int64, error) {
	likeQuery := "%" + escapeLike(query) + "%"
	matches := s.db.Model(&model.Post{}).Scopes(model.PublicPosts).
		Where("title LIKE ? OR content LIKE ?", likeQuery, likeQuery)
	return newestFirstPage(matches, page, pageSize)
}

// GetDrafts returns all draft posts for a user.
func (s *service) GetDrafts(userID uint) ([]model.Post, error) {
	var posts []model.Post
	if err := s.db.Where("user_id = ? AND status = ?", userID, model.PostStatusDraft).
		Preload("User").Preload("Tags").
		Order("updated_at DESC").
		Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}
