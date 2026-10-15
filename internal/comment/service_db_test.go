package comment

import (
	"errors"
	"slices"
	"testing"
	"time"

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

func TestCreateComment_Replies(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db)
	author := testutil.MakeUser(t, db, "author")
	reader := testutil.MakeUser(t, db, "reader")
	post := testutil.MakePost(t, db, author.ID)
	otherPost := testutil.MakePost(t, db, author.ID)
	parent := testutil.MakeComment(t, db, post.ID, author.ID, "원 댓글")
	elsewhere := testutil.MakeComment(t, db, otherPost.ID, author.ID, "다른 글 댓글")

	t.Run("같은 글의 댓글에 답글을 달고 작성자를 싣는다", func(t *testing.T) {
		c := &model.Comment{PostID: post.ID, UserID: reader.ID, ParentID: &parent.ID, Content: "답글"}
		if err := svc.CreateComment(c); err != nil {
			t.Fatalf("CreateComment: %v", err)
		}
		if c.User.Username != "reader" {
			t.Fatalf("User.Username = %q, want reader", c.User.Username)
		}
	})

	t.Run("다른 글의 댓글에는 답글을 달 수 없다", func(t *testing.T) {
		c := &model.Comment{PostID: post.ID, UserID: reader.ID, ParentID: &elsewhere.ID, Content: "엇갈린 답글"}
		if err := svc.CreateComment(c); !errors.Is(err, ErrParentNotFound) {
			t.Fatalf("err = %v, want ErrParentNotFound", err)
		}
	})

	t.Run("지워진 댓글에는 답글을 달 수 없다", func(t *testing.T) {
		gone := testutil.MakeComment(t, db, post.ID, author.ID, "지울 댓글")
		db.Delete(&model.Comment{}, gone.ID)
		c := &model.Comment{PostID: post.ID, UserID: reader.ID, ParentID: &gone.ID, Content: "늦은 답글"}
		if err := svc.CreateComment(c); !errors.Is(err, ErrParentNotFound) {
			t.Fatalf("err = %v, want ErrParentNotFound", err)
		}
	})

	t.Run("지워진 글과 초안에는 댓글을 달 수 없다", func(t *testing.T) {
		gonePost := testutil.MakePost(t, db, author.ID)
		db.Delete(&model.Post{}, gonePost.ID)
		draft := testutil.MakePost(t, db, author.ID, testutil.WithStatus("draft"))
		if err := svc.CreateComment(&model.Comment{PostID: gonePost.ID, UserID: reader.ID, Content: "x"}); !errors.Is(err, ErrPostNotFound) {
			t.Fatalf("지워진 글 err = %v, want ErrPostNotFound", err)
		}
		if err := svc.CreateComment(&model.Comment{PostID: draft.ID, UserID: reader.ID, Content: "x"}); !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("초안 err = %v, want ErrUnauthorized", err)
		}
	})
}

func TestGetCommentsByPostID(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db)
	author := testutil.MakeUser(t, db, "author")
	reader := testutil.MakeUser(t, db, "reader")
	post := testutil.MakePost(t, db, author.ID)
	other := testutil.MakePost(t, db, author.ID)
	base := time.Now().Add(-time.Hour)
	var want []uint
	for i, content := range []string{"첫째", "둘째", "셋째"} {
		c := testutil.MakeComment(t, db, post.ID, reader.ID, content)
		db.Model(c).UpdateColumn("created_at", base.Add(time.Duration(i)*time.Minute))
		want = append(want, c.ID)
	}
	gone := testutil.MakeComment(t, db, post.ID, reader.ID, "지운 댓글")
	db.Delete(&model.Comment{}, gone.ID)
	testutil.MakeComment(t, db, other.ID, reader.ID, "다른 글 댓글")

	ids := func(comments []model.Comment) []uint {
		out := make([]uint, len(comments))
		for i, c := range comments {
			out[i] = c.ID
		}
		return out
	}

	t.Run("이 글의 살아 있는 댓글을 오래된 순으로", func(t *testing.T) {
		comments, total, err := svc.GetCommentsByPostID(post.ID, nil, 50)
		if err != nil {
			t.Fatalf("GetCommentsByPostID: %v", err)
		}
		if total != 3 || !slices.Equal(ids(comments), want) {
			t.Fatalf("total=%d comments=%v, want 3 %v", total, ids(comments), want)
		}
		if comments[0].User.Username != "reader" {
			t.Fatal("댓글 작성자가 실리지 않았다")
		}
	})

	t.Run("limit 은 개수만 자르고 total 은 전체 수", func(t *testing.T) {
		comments, total, err := svc.GetCommentsByPostID(post.ID, nil, 2)
		if err != nil {
			t.Fatalf("GetCommentsByPostID: %v", err)
		}
		if total != 3 || !slices.Equal(ids(comments), want[:2]) {
			t.Fatalf("total=%d comments=%v, want 3 %v", total, ids(comments), want[:2])
		}
	})

	t.Run("숨은 글의 댓글은 작성자에게만", func(t *testing.T) {
		db.Model(post).UpdateColumn("is_public", false)
		defer db.Model(post).UpdateColumn("is_public", true)
		if _, _, err := svc.GetCommentsByPostID(post.ID, &reader.ID, 50); !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("다른 사용자 err = %v, want ErrUnauthorized", err)
		}
		if _, total, err := svc.GetCommentsByPostID(post.ID, &author.ID, 50); err != nil || total != 3 {
			t.Fatalf("작성자 total=%d err=%v, want 3, nil", total, err)
		}
	})

	t.Run("없는 글", func(t *testing.T) {
		if _, _, err := svc.GetCommentsByPostID(9999, nil, 50); !errors.Is(err, ErrPostNotFound) {
			t.Fatalf("err = %v, want ErrPostNotFound", err)
		}
	})
}

func TestUpdateAndDeleteComment_Missing(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db)
	user := testutil.MakeUser(t, db, "user")

	if _, err := svc.UpdateComment(9999, user.ID, "x"); !errors.Is(err, ErrCommentNotFound) {
		t.Fatalf("Update err = %v, want ErrCommentNotFound", err)
	}
	if err := svc.DeleteComment(9999, user.ID); !errors.Is(err, ErrCommentNotFound) {
		t.Fatalf("Delete err = %v, want ErrCommentNotFound", err)
	}
}
