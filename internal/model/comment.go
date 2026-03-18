package model

import (
	"time"

	"gorm.io/gorm"
)

type Comment struct {
	ID        uint           `gorm:"primaryKey;autoIncrement"`
	PostID    uint           `gorm:"not null;index:idx_comment_post_deleted,priority:1"`
	UserID    uint           `gorm:"not null;index"`
	ParentID  *uint          `gorm:"index"`
	Content   string         `gorm:"type:text;not null"`
	IsEdited  bool           `gorm:"default:false"`
	User      User           `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Replies   []Comment      `gorm:"foreignKey:ParentID"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index:idx_comment_post_deleted,priority:2" json:"-"`
}
