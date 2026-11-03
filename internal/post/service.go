package post

import (
	"errors"
	"log/slog"
	"strings"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/model"
	"unicode/utf8"

	"gorm.io/gorm"
)

var (
	ErrPostNotFound       = errors.New("post not found")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrNoFieldsToUpdate   = errors.New("no fields to update")
	ErrInvalidTag         = errors.New("invalid tag parameter")
	ErrInvalidSearchQuery = errors.New("invalid search query")
	ErrTagTooLong         = errors.New("tag name too long")
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
	IsLiked(postID uint, userID uint) (bool, error)
	GetDrafts(userID uint) ([]model.Post, error)
}

type service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

func withDetails(db *gorm.DB) *gorm.DB {
	return db.Preload("User").Preload("Tags").Preload("Series")
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

// syncTags 는 글의 태그를 쉼표로 나뉜 이름 그대로 맞추고, 없는 태그는 만든다.
func syncTags(tx *gorm.DB, post *model.Post, tagsStr string) error {
	tagNames := splitTags(tagsStr)
	tags := make([]model.Tag, 0, len(tagNames))
	for _, name := range tagNames {
		if utf8.RuneCountInString(name) > model.TagNameMaxRunes {
			return ErrTagTooLong
		}
		var tag model.Tag
		if err := tx.Where("name = ?", name).FirstOrCreate(&tag, model.Tag{Name: name}).Error; err != nil {
			return err
		}
		tags = append(tags, tag)
	}
	return tx.Model(post).Association("Tags").Replace(tags)
}

func (s *service) findPost(db *gorm.DB, postID uint) (*model.Post, error) {
	var post model.Post
	if err := db.First(&post, postID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}
	return &post, nil
}

func (s *service) findOwnPost(postID, userID uint) (*model.Post, error) {
	post, err := s.findPost(s.db, postID)
	if err != nil {
		return nil, err
	}
	if post.UserID != userID {
		return nil, ErrUnauthorized
	}
	return post, nil
}

func (s *service) CreatePost(post *model.Post) error {
	if post.Status == "" {
		post.Status = model.PostStatusPublished
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(post).Error; err != nil {
			return err
		}
		if post.TagsRaw != "" {
			if err := syncTags(tx, post, post.TagsRaw); err != nil {
				return err
			}
		}
		return tx.Scopes(withDetails).First(post, post.ID).Error
	})
}

func (s *service) GetPostByID(postID uint, userID *uint) (*model.Post, error) {
	post, err := s.findPost(s.db.Scopes(withDetails), postID)
	if err != nil {
		return nil, err
	}
	if !post.VisibleTo(userID) {
		return nil, ErrUnauthorized
	}

	// 조회수는 덜 세어도 글은 보여 준다.
	if err := s.db.Model(&model.Post{}).Where("id = ?", postID).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error; err != nil {
		slog.Warn("조회수 증가 실패", "post_id", postID, "error", err)
	} else {
		post.ViewCount++
	}
	return post, nil
}

func (s *service) UpdatePost(postID uint, userID uint, req *dto.UpdatePostRequest) (*model.Post, error) {
	post, err := s.findOwnPost(postID, userID)
	if err != nil {
		return nil, err
	}

	updates := changedPostColumns(req)
	if len(updates) == 0 {
		return nil, ErrNoFieldsToUpdate
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(post).Updates(updates).Error; err != nil {
			return err
		}
		if req.Tags != nil {
			if err := syncTags(tx, post, *req.Tags); err != nil {
				return err
			}
		}
		post, err = s.findPost(tx.Scopes(withDetails), postID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return post, nil
}

func changedPostColumns(req *dto.UpdatePostRequest) map[string]any {
	updates := map[string]any{}
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
	return updates
}

func (s *service) DeletePost(postID uint, userID uint) error {
	post, err := s.findOwnPost(postID, userID)
	if err != nil {
		return err
	}
	return s.db.Delete(post).Error
}
