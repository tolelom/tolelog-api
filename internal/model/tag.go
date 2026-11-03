package model

// TagNameMaxRunes 는 아래 name 칸의 size 와 같아야 한다. MySQL utf8mb4 VARCHAR 는 글자 수로 센다.
const TagNameMaxRunes = 50

type Tag struct {
	ID   uint   `gorm:"primaryKey;autoIncrement"`
	Name string `gorm:"size:50;uniqueIndex;not null"`
}
