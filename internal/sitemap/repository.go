package sitemap

import (
	"tolelom_api/internal/model"

	"gorm.io/gorm"
)

type gormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) GetPublicPostEntries() ([]Entry, error) {
	var rows []Entry
	err := r.db.Model(&model.Post{}).
		Select("id, title, updated_at").
		Where("is_public = ? AND deleted_at IS NULL AND status = ?", true, "published").
		Order("updated_at DESC").
		Find(&rows).Error
	return rows, err
}

// GetPublicAuthorEntries 는 공개 발행 글이 하나 이상 있는 작성자와 그 글의 마지막 수정 시각이다.
func (r *gormRepository) GetPublicAuthorEntries() ([]Entry, error) {
	var posts []Entry
	err := r.db.Model(&model.Post{}).
		Select("user_id AS id, updated_at").
		Where("is_public = ? AND deleted_at IS NULL AND status = ?", true, "published").
		Order("updated_at DESC").
		Find(&posts).Error
	if err != nil {
		return nil, err
	}
	seen := make(map[uint]bool)
	authors := make([]Entry, 0)
	for _, p := range posts {
		if !seen[p.ID] {
			seen[p.ID] = true
			authors = append(authors, p)
		}
	}
	return authors, nil
}

func (r *gormRepository) GetSeriesEntries() ([]Entry, error) {
	var rows []Entry
	err := r.db.Model(&model.Series{}).
		Select("id, updated_at").
		Where("deleted_at IS NULL").
		Order("updated_at DESC").
		Find(&rows).Error
	return rows, err
}
