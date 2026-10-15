package model

import (
	"time"

	"gorm.io/gorm"
)

type Post struct {
	ID        uint           `gorm:"primaryKey;autoIncrement"`
	Title     string         `gorm:"size:255;not null"`
	Content   string         `gorm:"type:longtext;not null"`
	UserID    uint           `gorm:"not null;index;index:idx_user_public,priority:1"`
	User      User           `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	IsPublic  bool           `gorm:"index;index:idx_user_public,priority:2"`
	Status    string         `gorm:"size:20;default:'published';index"`
	TagsRaw   string         `gorm:"column:tags;size:500;default:''"`
	Tags        []Tag          `gorm:"many2many:post_tags;" json:"-"`
	SeriesID    *uint          `gorm:"index"`
	SeriesOrder *int           `gorm:"default:null"`
	Series      *Series        `gorm:"foreignKey:SeriesID" json:"-"`
	ViewCount   uint           `gorm:"default:0"`
	LikeCount   uint           `gorm:"default:0"`
	CreatedAt   time.Time      `gorm:"autoCreateTime;index"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// VisibleTo - 공개된 발행 글은 누구나, 그 외(비공개·draft)는 작성자만 볼 수 있다. 목록 쿼리와 같은 기준.
func (p *Post) VisibleTo(userID *uint) bool {
	if userID != nil && *userID == p.UserID {
		return true
	}
	return p.IsPublic && p.Status == "published"
}
