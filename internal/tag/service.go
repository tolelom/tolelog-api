package tag

import (
	"tolelom_api/internal/dto"

	"gorm.io/gorm"
)

type Service interface {
	GetTags(sort string, limit int) (*dto.TagListResponse, error)
}

type service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

type tagRow struct {
	Name      string
	PostCount int64
}

func (s *service) GetTags(sort string, limit int) (*dto.TagListResponse, error) {
	orderClause := "post_count DESC"
	if sort == "name" {
		orderClause = "tags.name ASC"
	}

	var rows []tagRow
	err := s.db.Table("tags").
		Select("tags.name, COUNT(DISTINCT post_tags.post_id) as post_count").
		Joins("JOIN post_tags ON post_tags.tag_id = tags.id").
		Joins("JOIN posts ON posts.id = post_tags.post_id").
		Where("posts.is_public = ? AND posts.deleted_at IS NULL", true).
		Group("tags.id").
		Order(orderClause).
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	tags := make([]dto.TagResponse, len(rows))
	for i, r := range rows {
		tags[i] = dto.TagResponse{
			Name:      r.Name,
			PostCount: r.PostCount,
		}
	}

	return &dto.TagListResponse{
		Tags:  tags,
		Total: int64(len(tags)),
	}, nil
}
