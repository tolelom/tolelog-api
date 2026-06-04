package comment

import (
	"errors"
	"testing"

	"tolelom_api/internal/model"
	"tolelom_api/internal/testutil"
)

// 경로 8: 비공개 글에 타인 댓글 차단, 글 작성자 본인은 가능
func TestCreateComment_PrivatePostAccessControl(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db)

	owner := testutil.MakeUser(t, db, "owner")
	other := testutil.MakeUser(t, db, "other")
	private := testutil.MakePost(t, db, owner.ID, testutil.WithPrivate())

	t.Run("타인의 비공개 글 댓글 차단", func(t *testing.T) {
		c := &model.Comment{PostID: private.ID, UserID: other.ID, Content: "몰래 댓글"}
		err := svc.CreateComment(c)
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("err = %v, want ErrUnauthorized", err)
		}
	})

	t.Run("글 작성자 본인은 비공개 글에 댓글 가능", func(t *testing.T) {
		c := &model.Comment{PostID: private.ID, UserID: owner.ID, Content: "셀프 댓글"}
		if err := svc.CreateComment(c); err != nil {
			t.Fatalf("본인 댓글 실패: %v", err)
		}
		if c.ID == 0 {
			t.Fatal("comment ID 미할당")
		}
	})

	t.Run("공개 글에는 누구나 댓글 가능", func(t *testing.T) {
		public := testutil.MakePost(t, db, owner.ID)
		c := &model.Comment{PostID: public.ID, UserID: other.ID, Content: "공개 댓글"}
		if err := svc.CreateComment(c); err != nil {
			t.Fatalf("공개 글 댓글 실패: %v", err)
		}
	})
}

// 경로 9: 댓글 작성자 본인만 삭제 가능 (현재 구현 — 글 작성자 권한 없음)
// 서브테스트는 순차 시나리오 (차단 → 삭제) — 단독 실행 불가.
func TestDeleteComment_AuthorOnly(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db)

	postOwner := testutil.MakeUser(t, db, "post-owner")
	commenter := testutil.MakeUser(t, db, "commenter")
	post := testutil.MakePost(t, db, postOwner.ID)
	comment := testutil.MakeComment(t, db, post.ID, commenter.ID, "지워질 댓글")

	t.Run("글 작성자도 타인 댓글 삭제 불가 (현재 구현)", func(t *testing.T) {
		err := svc.DeleteComment(comment.ID, postOwner.ID)
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("err = %v, want ErrUnauthorized", err)
		}
	})

	t.Run("작성자 본인은 삭제 가능 (soft delete)", func(t *testing.T) {
		if err := svc.DeleteComment(comment.ID, commenter.ID); err != nil {
			t.Fatalf("본인 삭제 실패: %v", err)
		}
		var count int64
		if err := db.Model(&model.Comment{}).Where("id = ?", comment.ID).Count(&count).Error; err != nil {
			t.Fatalf("count 쿼리 실패: %v", err)
		}
		if count != 0 {
			t.Fatal("soft delete 후에도 기본 쿼리에 노출됨")
		}
	})
}

// 경로 10: 작성자 본인만 수정 가능
// 서브테스트는 순차 시나리오 (차단 → 수정) — 단독 실행 불가.
func TestUpdateComment_AuthorOnly(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db)

	commenter := testutil.MakeUser(t, db, "commenter")
	intruder := testutil.MakeUser(t, db, "intruder")
	post := testutil.MakePost(t, db, commenter.ID)
	comment := testutil.MakeComment(t, db, post.ID, commenter.ID, "원본")

	t.Run("타인 수정 차단", func(t *testing.T) {
		_, err := svc.UpdateComment(comment.ID, intruder.ID, "변조")
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("err = %v, want ErrUnauthorized", err)
		}
		var reloaded model.Comment
		if err := db.First(&reloaded, comment.ID).Error; err != nil {
			t.Fatalf("reload 실패: %v", err)
		}
		if reloaded.Content != "원본" {
			t.Fatalf("차단됐는데 DB 변경됨: %q", reloaded.Content)
		}
	})

	t.Run("본인 수정 가능 + IsEdited 마킹", func(t *testing.T) {
		updated, err := svc.UpdateComment(comment.ID, commenter.ID, "수정본")
		if err != nil {
			t.Fatalf("본인 수정 실패: %v", err)
		}
		if updated.Content != "수정본" || !updated.IsEdited {
			t.Fatalf("content=%q isEdited=%v, want 수정본/true", updated.Content, updated.IsEdited)
		}
	})
}
