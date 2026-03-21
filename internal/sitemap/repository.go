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
		Select("id, updated_at").
		Where("is_public = ? AND deleted_at IS NULL AND status = ?", true, "published").
		Order("updated_at DESC").
		Find(&rows).Error
	return rows, err
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
