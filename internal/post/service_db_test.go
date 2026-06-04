package post

import (
	"errors"
	"testing"

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
