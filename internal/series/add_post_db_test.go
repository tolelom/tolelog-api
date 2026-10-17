package series

import (
	"testing"

	"gorm.io/gorm"

	"tolelom_api/internal/model"
	"tolelom_api/internal/testutil"
)

// AddPostToSeries 가 글을 읽은 뒤 쓰기 전에 다른 요청이 글을 고쳐도, 시리즈 칸만
// 쓰고 그 변경을 덮지 않는다.
func TestAddPostToSeries_DoesNotOverwriteConcurrentPostEdit(t *testing.T) {
	db := testutil.SetupDB(t)
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(db)
	owner := testutil.MakeUser(t, db, "owner")
	post := testutil.MakePost(t, db, owner.ID, testutil.WithTitle("원래 제목"))
	s := &model.Series{Title: "시리즈", UserID: owner.ID}
	if err := db.Create(s).Error; err != nil {
		t.Fatal(err)
	}

	fired := false
	err := db.Callback().Update().Before("gorm:update").Register("test:concurrent_edit", func(tx *gorm.DB) {
		if fired || tx.Statement.Table != "posts" {
			return
		}
		fired = true
		// 별도 세션: 같은 tx 로 Exec 하면 진행 중인 구문의 조건이 지워진다.
		tx.Session(&gorm.Session{NewDB: true}).Exec("UPDATE posts SET title = ?, content = ? WHERE id = ?", "다른 요청이 고친 제목", "다른 요청이 고친 본문", post.ID)
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.AddPostToSeries(s.ID, post.ID, 0, owner.ID); err != nil {
		t.Fatalf("AddPostToSeries: %v", err)
	}
	if !fired {
		t.Fatal("concurrent edit was not injected")
	}

	var got model.Post
	if err := db.First(&got, post.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.Title != "다른 요청이 고친 제목" || got.Content != "다른 요청이 고친 본문" {
		t.Errorf("title=%q content=%q, want the concurrent edit kept", got.Title, got.Content)
	}
	if got.SeriesID == nil || *got.SeriesID != s.ID || got.SeriesOrder == nil || *got.SeriesOrder != 1 {
		t.Errorf("series_id=%v series_order=%v, want %d/1", got.SeriesID, got.SeriesOrder, s.ID)
	}
}
