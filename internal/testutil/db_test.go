package testutil

import (
	"testing"

	"tolelom_api/internal/model"
)

func TestSetupDB_MigratesAllModels(t *testing.T) {
	db := SetupDB(t)

	// 전 모델 테이블이 존재하고 INSERT 가능해야 한다
	u := &model.User{Username: "smoke", PasswordHash: "x"}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("user insert 실패: %v", err)
	}
	tag := &model.Tag{Name: "go"}
	if err := db.Create(tag).Error; err != nil {
		t.Fatalf("tag insert 실패: %v", err)
	}
	s := &model.Series{Title: "s", UserID: u.ID}
	if err := db.Create(s).Error; err != nil {
		t.Fatalf("series insert 실패: %v", err)
	}
	p := &model.Post{Title: "t", Content: "c", UserID: u.ID, SeriesID: &s.ID}
	if err := db.Create(p).Error; err != nil {
		t.Fatalf("post insert 실패: %v", err)
	}
	if err := db.Model(p).Association("Tags").Append(tag); err != nil {
		t.Fatalf("post-tag M:N 연결 실패: %v", err)
	}
	c := &model.Comment{PostID: p.ID, UserID: u.ID, Content: "hi"}
	if err := db.Create(c).Error; err != nil {
		t.Fatalf("comment insert 실패: %v", err)
	}
	l := &model.PostLike{PostID: p.ID, UserID: u.ID}
	if err := db.Create(l).Error; err != nil {
		t.Fatalf("post_like insert 실패: %v", err)
	}
}

func TestSetupDB_IsolatedBetweenCalls(t *testing.T) {
	db1 := SetupDB(t)
	db2 := SetupDB(t)

	db1.Create(&model.User{Username: "only-in-db1", PasswordHash: "x"})

	var count int64
	db2.Model(&model.User{}).Count(&count)
	if count != 0 {
		t.Fatalf("DB 격리 실패: db2에 %d명 존재", count)
	}
}
