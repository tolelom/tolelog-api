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
		Scopes(model.PublicPosts).
		Order("updated_at DESC").
		Find(&rows).Error
	return rows, err
}

// GetPublicAuthorEntries 는 공개 발행 글이 하나 이상 있는 작성자와 그 글의 마지막 수정 시각이다.
func (r *gormRepository) GetPublicAuthorEntries() ([]Entry, error) {
	var posts []Entry
	err := r.db.Model(&model.Post{}).
		Select("user_id AS id, updated_at").
		Scopes(model.PublicPosts).
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

// GetSeriesEntries 는 공개 발행 글이 하나 이상 있는 시리즈다. 시리즈 페이지는 공개 글만
// 보여 주므로 빈 시리즈나 초안·비공개 글만 있는 시리즈는 넣지 않는다.
func (r *gormRepository) GetSeriesEntries() ([]Entry, error) {
	var rows []Entry
	hasPublicPost := r.db.Model(&model.Post{}).
		Select("1").
		Where("posts.series_id = series.id").
		Scopes(model.PublicPosts)
	err := r.db.Model(&model.Series{}).
		Select("id, updated_at").
		Where("EXISTS (?)", hasPublicPost).
		Order("updated_at DESC").
		Find(&rows).Error
	return rows, err
}
