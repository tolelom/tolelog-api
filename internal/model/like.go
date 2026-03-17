package model

import "time"

type PostLike struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	PostID    uint      `gorm:"not null;uniqueIndex:idx_post_user"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_post_user"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
