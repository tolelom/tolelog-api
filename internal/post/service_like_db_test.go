package post

import (
	"testing"

	"gorm.io/gorm"

	"tolelom_api/internal/model"
	"tolelom_api/internal/testutil"
)

// 같은 사용자의 요청 두 개가 겹친 상황을 결정적으로 만든다. 이 요청이 좋아요 행을
// 읽은 뒤, 쓰기 직전에 다른 요청의 변경이 커밋된 것처럼 once 를 한 번 실행한다.
// (테스트 DB 는 커넥션이 1개라 진짜 동시 실행으로는 이 순서가 나오지 않는다.)
func injectBefore(t *testing.T, db *gorm.DB, kind string, once func(tx *gorm.DB)) {
	t.Helper()
	fired := false
	cb := func(tx *gorm.DB) {
		if fired || tx.Statement.Table != "post_likes" {
			return
		}
		fired = true
		// 별도 세션: 같은 tx 로 Exec 하면 진행 중인 구문의 조건이 지워진다.
		once(tx.Session(&gorm.Session{NewDB: true}))
	}
	var err error
	switch kind {
	case "delete":
		err = db.Callback().Delete().Before("gorm:delete").Register("test:concurrent", cb)
	case "create":
		err = db.Callback().Create().Before("gorm:create").Register("test:concurrent", cb)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func likeState(t *testing.T, db *gorm.DB, postID uint) (likeCount uint, rows int64) {
	t.Helper()
	var p model.Post
	if err := db.Select("like_count").First(&p, postID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.PostLike{}).Where("post_id = ?", postID).Count(&rows).Error; err != nil {
		t.Fatal(err)
	}
	return p.LikeCount, rows
}

func TestToggleLike_ConcurrentDoubleUnlikeKeepsCount(t *testing.T) {
	db := testutil.SetupDB(t)
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(db)
	owner := testutil.MakeUser(t, db, "owner")
	a := testutil.MakeUser(t, db, "a")
	b := testutil.MakeUser(t, db, "b")
	p := testutil.MakePost(t, db, owner.ID)
	for _, u := range []uint{a.ID, b.ID} {
		if _, _, err := svc.ToggleLike(p.ID, u); err != nil {
			t.Fatal(err)
		}
	}

	// A 의 다른 취소 요청이 먼저 커밋됐다.
	injectBefore(t, db, "delete", func(tx *gorm.DB) {
		tx.Exec("DELETE FROM post_likes WHERE post_id = ? AND user_id = ?", p.ID, a.ID)
		tx.Exec("UPDATE posts SET like_count = like_count - 1 WHERE id = ?", p.ID)
	})
	liked, count, err := svc.ToggleLike(p.ID, a.ID)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	gotCount, rows := likeState(t, db, p.ID)
	if liked || count != 1 || gotCount != 1 || rows != 1 {
		t.Fatalf("liked=%v returned=%d stored=%d rows=%d, want false/1/1/1 (B 의 좋아요만 남음)", liked, count, gotCount, rows)
	}
}

func TestToggleLike_ConcurrentDoubleLikeIsIdempotent(t *testing.T) {
	db := testutil.SetupDB(t)
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(db)
	owner := testutil.MakeUser(t, db, "owner")
	a := testutil.MakeUser(t, db, "a")
	p := testutil.MakePost(t, db, owner.ID)

	// A 의 다른 좋아요 요청이 먼저 커밋됐다.
	injectBefore(t, db, "create", func(tx *gorm.DB) {
		tx.Exec("INSERT INTO post_likes (post_id, user_id, created_at) VALUES (?, ?, CURRENT_TIMESTAMP)", p.ID, a.ID)
		tx.Exec("UPDATE posts SET like_count = like_count + 1 WHERE id = ?", p.ID)
	})
	liked, count, err := svc.ToggleLike(p.ID, a.ID)
	if err != nil {
		t.Fatalf("err = %v, want nil (중복 좋아요는 이미 좋아요 상태로 본다)", err)
	}
	gotCount, rows := likeState(t, db, p.ID)
	if !liked || count != 1 || gotCount != 1 || rows != 1 {
		t.Fatalf("liked=%v returned=%d stored=%d rows=%d, want true/1/1/1", liked, count, gotCount, rows)
	}
}

func TestToggleLike_UnlikeNeverGoesBelowZero(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db)
	owner := testutil.MakeUser(t, db, "owner")
	a := testutil.MakeUser(t, db, "a")
	p := testutil.MakePost(t, db, owner.ID)
	if _, _, err := svc.ToggleLike(p.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	// 이미 틀어진 수치(행은 1개인데 0)에서 취소해도 음수가 되지 않는다.
	if err := db.Exec("UPDATE posts SET like_count = 0 WHERE id = ?", p.ID).Error; err != nil {
		t.Fatal(err)
	}
	if _, count, err := svc.ToggleLike(p.ID, a.ID); err != nil || count != 0 {
		t.Fatalf("count=%d err=%v, want 0/nil", count, err)
	}
}
