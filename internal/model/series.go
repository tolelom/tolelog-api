package model

import (
	"time"

	"gorm.io/gorm"
)

type Series struct {
	ID          uint           `gorm:"primaryKey;autoIncrement"`
	Title       string         `gorm:"size:255;not null"`
	Description string         `gorm:"type:text"`
	UserID      uint           `gorm:"not null;index"`
	User        User           `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Posts       []Post         `gorm:"foreignKey:SeriesID" json:"-"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
