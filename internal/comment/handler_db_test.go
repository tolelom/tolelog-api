package comment

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tolelom_api/internal/model"
	"tolelom_api/internal/testutil"
)

// 실제 서비스·DB(외래 키 켬)로 경로의 글 ID와 댓글 소속, 글 가시성을 확인한다.
func TestCommentRoutes_PostIDAndVisibility(t *testing.T) {
	db := testutil.SetupDB(t)
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(db)

	author := testutil.MakeUser(t, db, "author")
	commenter := testutil.MakeUser(t, db, "commenter")
	postA := testutil.MakePost(t, db, author.ID)
	postB := testutil.MakePost(t, db, author.ID)
	app := setupAuthApp(svc, commenter.ID)

	do := func(method, path, body string) int {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		return resp.StatusCode
	}
	contentOf := func(id uint) string {
		t.Helper()
		var c model.Comment
		if err := db.Unscoped().First(&c, id).Error; err != nil {
			t.Fatal(err)
		}
		return c.Content
	}

	t.Run("다른 글 경로로 내 댓글 수정은 404이고 내용이 그대로다", func(t *testing.T) {
		c := testutil.MakeComment(t, db, postA.ID, commenter.ID, "원래 내용")
		got := do(http.MethodPut, fmt.Sprintf("/posts/%d/comments/%d", postB.ID, c.ID), `{"content":"바뀐 내용"}`)
		if got != http.StatusNotFound {
			t.Errorf("status = %d, want 404", got)
		}
		if contentOf(c.ID) != "원래 내용" {
			t.Errorf("content = %q, want unchanged", contentOf(c.ID))
		}
	})

	t.Run("다른 글 경로로 내 댓글 삭제는 404이고 댓글이 남는다", func(t *testing.T) {
		c := testutil.MakeComment(t, db, postA.ID, commenter.ID, "남을 댓글")
		got := do(http.MethodDelete, fmt.Sprintf("/posts/%d/comments/%d", postB.ID, c.ID), "")
		if got != http.StatusNotFound {
			t.Errorf("status = %d, want 404", got)
		}
		var n int64
		db.Model(&model.Comment{}).Where("id = ?", c.ID).Count(&n)
		if n != 1 {
			t.Errorf("comment deleted through another post's path")
		}
	})

	t.Run("맞는 글 경로로는 수정·삭제된다", func(t *testing.T) {
		c := testutil.MakeComment(t, db, postA.ID, commenter.ID, "고칠 댓글")
		if got := do(http.MethodPut, fmt.Sprintf("/posts/%d/comments/%d", postA.ID, c.ID), `{"content":"고침"}`); got != http.StatusOK {
			t.Fatalf("update status = %d, want 200", got)
		}
		if contentOf(c.ID) != "고침" {
			t.Errorf("content = %q, want 고침", contentOf(c.ID))
		}
		if got := do(http.MethodDelete, fmt.Sprintf("/posts/%d/comments/%d", postA.ID, c.ID), ""); got != http.StatusNoContent {
			t.Errorf("delete status = %d, want 204", got)
		}
	})

	t.Run("볼 수 없게 된 글의 내 댓글은 수정할 수 없고 삭제는 된다", func(t *testing.T) {
		hidden := testutil.MakePost(t, db, author.ID)
		c := testutil.MakeComment(t, db, hidden.ID, commenter.ID, "공개였을 때 쓴 댓글")
		if err := db.Model(&model.Post{}).Where("id = ?", hidden.ID).Update("is_public", false).Error; err != nil {
			t.Fatal(err)
		}
		if got := do(http.MethodPut, fmt.Sprintf("/posts/%d/comments/%d", hidden.ID, c.ID), `{"content":"몰래 고침"}`); got != http.StatusForbidden {
			t.Errorf("update status = %d, want 403", got)
		}
		if contentOf(c.ID) != "공개였을 때 쓴 댓글" {
			t.Errorf("content = %q, want unchanged", contentOf(c.ID))
		}
		if got := do(http.MethodDelete, fmt.Sprintf("/posts/%d/comments/%d", hidden.ID, c.ID), ""); got != http.StatusNoContent {
			t.Errorf("delete status = %d, want 204 (own comment stays deletable)", got)
		}
	})
}
