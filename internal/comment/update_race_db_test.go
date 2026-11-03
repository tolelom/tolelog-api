package comment

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"tolelom_api/internal/model"
	"tolelom_api/internal/testutil"
)

// 수정 요청이 댓글을 읽은 뒤 쓰기 직전에, 같은 작성자의 삭제 요청이 커밋된 상황을 만든다.
func TestUpdateComment_DoesNotResurrectConcurrentlyDeletedComment(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db)
	u := testutil.MakeUser(t, db, "alice")
	p := testutil.MakePost(t, db, u.ID)
	c := testutil.MakeComment(t, db, p.ID, u.ID, "원래 댓글")

	fired := false
	err := db.Callback().Update().Before("gorm:update").Register("test:concurrent_delete", func(tx *gorm.DB) {
		if fired || tx.Statement.Table != "comments" {
			return
		}
		fired = true
		tx.Session(&gorm.Session{NewDB: true}).
			Exec("UPDATE comments SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?", c.ID)
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.UpdateComment(p.ID, c.ID, u.ID, "고친 댓글")
	if !fired {
		t.Fatal("겹친 삭제가 주입되지 않았다")
	}
	if !errors.Is(err, ErrCommentNotFound) {
		t.Errorf("err = %v, want ErrCommentNotFound", err)
	}
	var visible int64
	db.Model(&model.Comment{}).Where("id = ?", c.ID).Count(&visible)
	if visible != 0 {
		t.Fatal("삭제된 댓글이 수정 요청으로 되살아났다")
	}
}
