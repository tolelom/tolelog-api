package model

import "time"

type User struct {
	ID           uint   `gorm:"primaryKey;autoIncrement"`
	Username     string `gorm:"size:100;uniqueIndex;not null"`
	PasswordHash string `gorm:"column:password;not null"`
	AvatarURL    string `gorm:"size:500;default:''"`
	CreatedAt    time.Time
	LastLogin    time.Time
	TokenVersion int `gorm:"default:0"`
}
