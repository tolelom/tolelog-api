package model

import (
	"time"

	"gorm.io/gorm"
)

type Post struct {
	ID        uint           `gorm:"primaryKey;autoIncrement"`
	Title     string         `gorm:"size:255;not null"`
	Content   string         `gorm:"type:longtext;not null"`
	UserID    uint           `gorm:"not null;index"`
	User      User           `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	IsPublic  bool           `gorm:"default:true"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
