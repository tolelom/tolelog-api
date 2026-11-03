package model

import (
	"time"

	"gorm.io/gorm"
)

type Post struct {
	ID          uint           `gorm:"primaryKey;autoIncrement"`
	Title       string         `gorm:"size:255;not null"`
	Content     string         `gorm:"type:longtext;not null"`
	UserID      uint           `gorm:"not null;index;index:idx_user_public,priority:1"`
	User        User           `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	IsPublic    bool           `gorm:"index;index:idx_user_public,priority:2"`
	Status      PostStatus     `gorm:"size:20;default:'published';index"`
	TagsRaw     string         `gorm:"column:tags;size:500;default:''"`
	Tags        []Tag          `gorm:"many2many:post_tags;" json:"-"`
	SeriesID    *uint          `gorm:"index"`
	SeriesOrder *int           `gorm:"default:null"`
	Series      *Series        `gorm:"foreignKey:SeriesID" json:"-"`
	ViewCount   uint           `gorm:"default:0"`
	LikeCount   uint           `gorm:"default:0"`
	CreatedAt   time.Time      `gorm:"autoCreateTime;index"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// VisibleTo 와 PublicPosts 는 같은 기준이어야 한다: 공개 발행 글은 누구나, 나머지는 작성자만 본다.
func (p *Post) VisibleTo(userID *uint) bool {
	if userID != nil && *userID == p.UserID {
		return true
	}
	return p.IsPublic && p.Status == PostStatusPublished
}

// OrderInSeries is 0 for a post without an order.
func (p *Post) OrderInSeries() int {
	if p.SeriesOrder == nil {
		return 0
	}
	return *p.SeriesOrder
}

func PublicPosts(db *gorm.DB) *gorm.DB {
	return db.Where("posts.is_public = ? AND posts.status = ?", true, PostStatusPublished)
}
