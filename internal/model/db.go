package model

import "gorm.io/gorm"

var DB *gorm.DB

func SetDB(database *gorm.DB) {
	DB = database
}

func GetDB() *gorm.DB {
	return DB
}
