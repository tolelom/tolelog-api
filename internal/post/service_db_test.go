package post

import (
	"errors"
	"testing"

	"tolelom_api/internal/dto"
	"tolelom_api/internal/model"
	"tolelom_api/internal/testutil"
)

// 경로 1: is_public=false 글을 비로그인/타인이 조회 시 차단
// 경로 2: 본인은 자기 비공개 글 조회 가능
// GetPostByID 호출은 view_count를 +1 한다 — 이 테스트는 count를 단언하지 않는다.
func TestGetPostByID_PrivatePostAccessControl(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db, nil) // Redis nil — 서비스가 nil 가드 보유

	owner := testutil.MakeUser(t, db, "owner")
	other := testutil.MakeUser(t, db, "other")
	private := testutil.MakePost(t, db, owner.ID, testutil.WithPrivate())
	public := testutil.MakePost(t, db, owner.ID)

	t.Run("비로그인(userID=nil) 조회 차단", func(t *testing.T) {
		_, err := svc.GetPostByID(private.ID, nil)
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("err = %v, want ErrUnauthorized", err)
		}
	})

	t.Run("타인 조회 차단", func(t *testing.T) {
		_, err := svc.GetPostByID(private.ID, &other.ID)
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("err = %v, want ErrUnauthorized", err)
		}
	})

	t.Run("본인은 조회 가능", func(t *testing.T) {
		got, err := svc.GetPostByID(private.ID, &owner.ID)
		if err != nil {
			t.Fatalf("본인 조회 실패: %v", err)
		}
		if got.ID != private.ID {
			t.Fatalf("got.ID = %d, want %d", got.ID, private.ID)
		}
	})

	t.Run("공개 글은 비로그인도 조회 가능", func(t *testing.T) {
		got, err := svc.GetPostByID(public.ID, nil)
		if err != nil {
			t.Fatalf("공개 글 조회 실패: %v", err)
		}
		if got.ID != public.ID {
			t.Fatalf("got.ID = %d, want %d", got.ID, public.ID)
		}
	})
}

func strPtr(s string) *string { return &s }

// 경로 3: 본인 글만 수정 가능
func TestUpdatePost_OwnershipEnforced(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db, nil)

	owner := testutil.MakeUser(t, db, "owner")
	attacker := testutil.MakeUser(t, db, "attacker")
	post := testutil.MakePost(t, db, owner.ID, testutil.WithTitle("원래 제목"))

	t.Run("타인 수정 차단", func(t *testing.T) {
		_, err := svc.UpdatePost(post.ID, attacker.ID, &dto.UpdatePostRequest{Title: strPtr("해킹")})
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("err = %v, want ErrUnauthorized", err)
		}
		// DB가 실제로 변경되지 않았는지 확인
		var reloaded model.Post
		if err := db.First(&reloaded, post.ID).Error; err != nil {
			t.Fatalf("reload 실패: %v", err)
		}
		if reloaded.Title != "원래 제목" {
			t.Fatalf("차단됐는데 DB 변경됨: %q", reloaded.Title)
		}
	})

	t.Run("본인 수정 가능", func(t *testing.T) {
		updated, err := svc.UpdatePost(post.ID, owner.ID, &dto.UpdatePostRequest{Title: strPtr("새 제목")})
		if err != nil {
			t.Fatalf("본인 수정 실패: %v", err)
		}
		if updated.Title != "새 제목" {
			t.Fatalf("title = %q, want 새 제목", updated.Title)
		}
	})
}

// 경로 4: 본인 글만 삭제 가능
func TestDeletePost_OwnershipEnforced(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db, nil)

	owner := testutil.MakeUser(t, db, "owner")
	attacker := testutil.MakeUser(t, db, "attacker")
	post := testutil.MakePost(t, db, owner.ID)

	t.Run("타인 삭제 차단", func(t *testing.T) {
		err := svc.DeletePost(post.ID, attacker.ID)
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("err = %v, want ErrUnauthorized", err)
		}
	})

	t.Run("본인 삭제 가능 (soft delete)", func(t *testing.T) {
		if err := svc.DeletePost(post.ID, owner.ID); err != nil {
			t.Fatalf("본인 삭제 실패: %v", err)
		}
		var count int64
		db.Model(&model.Post{}).Where("id = ?", post.ID).Count(&count)
		if count != 0 {
			t.Fatal("soft delete 후에도 기본 쿼리에 노출됨")
		}
	})

	t.Run("삭제된 글 재삭제는 NotFound", func(t *testing.T) {
		err := svc.DeletePost(post.ID, owner.ID)
		if !errors.Is(err, ErrPostNotFound) {
			t.Fatalf("err = %v, want ErrPostNotFound", err)
		}
	})
}
