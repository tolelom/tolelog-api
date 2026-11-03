package post

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"tolelom_api/internal/dto"
	"tolelom_api/internal/testutil"
)

// 수정 요청이 글을 읽은 뒤 쓰기 직전에, 같은 작성자의 삭제 요청이 커밋된 상황을 만든다.
func TestUpdatePost_ConcurrentlyDeletedPostIsNotFound(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db)
	u := testutil.MakeUser(t, db, "alice")
	p := testutil.MakePost(t, db, u.ID)

	fired := false
	err := db.Callback().Update().Before("gorm:update").Register("test:concurrent_delete", func(tx *gorm.DB) {
		if fired || tx.Statement.Table != "posts" {
			return
		}
		fired = true
		tx.Session(&gorm.Session{NewDB: true}).
			Exec("UPDATE posts SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?", p.ID)
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.UpdatePost(p.ID, u.ID, &dto.UpdatePostRequest{Title: strPtr("고친 제목")})
	if !fired {
		t.Fatal("겹친 삭제가 주입되지 않았다")
	}
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("err = %v, want ErrPostNotFound", err)
	}
}
