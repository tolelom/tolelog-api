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

// 글자·숫자·공백과 흔한 문장부호만 받는다. LIKE 의 % 는 받지 않고, _ 는 escapeLike 가 이스케이프한다.
var validSearchPattern = regexp.MustCompile(`^[\p{L}\p{N}\s\-_\.\,\!\?\:\;\'\"\(\)]+$`)

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

// SanitizeTag 는 빈 태그를 필터 없음으로 보고 오류 없이 "" 를 돌려준다.
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

// escapeLike 는 MySQL LIKE 의 기본 이스케이프 문자인 \ 를 쓴다.
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
