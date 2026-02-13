package model

import "time"

type User struct {
	ID           uint   `gorm:"primaryKey;autoIncrement"`
	Username     string `gorm:"size:100;uniqueIndex;not null"`
	PasswordHash string `gorm:"column:password;not null"`
	CreatedAt    time.Time
	LastLogin    time.Time
}
