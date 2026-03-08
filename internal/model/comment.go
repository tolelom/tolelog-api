package model

import (
	"time"

	"gorm.io/gorm"
)

type Comment struct {
	ID        uint           `gorm:"primaryKey;autoIncrement"`
	PostID    uint           `gorm:"not null;index:idx_comment_post_deleted,priority:1"`
	UserID    uint           `gorm:"not null;index"`
	Content   string         `gorm:"type:text;not null"`
	User      User           `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index:idx_comment_post_deleted,priority:2" json:"-"`
}
