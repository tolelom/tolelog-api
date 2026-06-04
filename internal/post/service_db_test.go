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
// 서브테스트는 순차 시나리오 (차단 → 수정) — 단독 실행 불가.
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
// 서브테스트는 순차 시나리오 (차단 → 삭제 → 재삭제) — 단독 실행 불가.
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
		if err := db.Model(&model.Post{}).Where("id = ?", post.ID).Count(&count).Error; err != nil {
			t.Fatalf("count 쿼리 실패: %v", err)
		}
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

// 경로 5: 타인 프로필 조회 시 is_public=true && published 만, 본인은 전부
func TestGetUserPosts_VisibilityByViewer(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db, nil)

	owner := testutil.MakeUser(t, db, "owner")
	viewer := testutil.MakeUser(t, db, "viewer")
	testutil.MakePost(t, db, owner.ID)                               // 공개+published
	testutil.MakePost(t, db, owner.ID, testutil.WithPrivate())       // 비공개
	testutil.MakePost(t, db, owner.ID, testutil.WithStatus("draft")) // 초안

	t.Run("비로그인은 공개 글만", func(t *testing.T) {
		posts, total, err := svc.GetUserPosts(owner.ID, nil, 1, 10, "")
		if err != nil {
			t.Fatalf("조회 실패: %v", err)
		}
		if total != 1 || len(posts) != 1 {
			t.Fatalf("total=%d len=%d, want 1/1", total, len(posts))
		}
	})

	t.Run("타인은 공개 글만", func(t *testing.T) {
		posts, total, err := svc.GetUserPosts(owner.ID, &viewer.ID, 1, 10, "")
		if err != nil {
			t.Fatalf("조회 실패: %v", err)
		}
		if total != 1 || len(posts) != 1 {
			t.Fatalf("total=%d len=%d, want 1/1", total, len(posts))
		}
	})

	t.Run("본인은 전부", func(t *testing.T) {
		posts, total, err := svc.GetUserPosts(owner.ID, &owner.ID, 1, 10, "")
		if err != nil {
			t.Fatalf("조회 실패: %v", err)
		}
		if total != 3 || len(posts) != 3 {
			t.Fatalf("total=%d len=%d, want 3/3", total, len(posts))
		}
	})
}

// 경로 6: 본인 초안만 조회
func TestGetDrafts_OnlyOwnDrafts(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db, nil)

	me := testutil.MakeUser(t, db, "me")
	someone := testutil.MakeUser(t, db, "someone")
	testutil.MakePost(t, db, me.ID, testutil.WithStatus("draft"), testutil.WithTitle("내 초안"))
	testutil.MakePost(t, db, me.ID)                                    // published — 초안 아님
	testutil.MakePost(t, db, someone.ID, testutil.WithStatus("draft")) // 남의 초안

	drafts, err := svc.GetDrafts(me.ID)
	if err != nil {
		t.Fatalf("조회 실패: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("len=%d, want 1", len(drafts))
	}
	if drafts[0].Title != "내 초안" {
		t.Fatalf("title=%q, want 내 초안", drafts[0].Title)
	}
}

// 경로 7: 같은 유저 두 번 토글 시 liked 상태/count 증감 정상
// 순차 시나리오 (좋아요 → 취소 → 다른 유저 좋아요) — 단계별 상태 누적에 의존.
func TestToggleLike_TogglesAndCounts(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db, nil)

	author := testutil.MakeUser(t, db, "author")
	fan := testutil.MakeUser(t, db, "fan")
	post := testutil.MakePost(t, db, author.ID)

	// 1차 토글: 좋아요
	liked, count, err := svc.ToggleLike(post.ID, fan.ID)
	if err != nil {
		t.Fatalf("1차 토글 실패: %v", err)
	}
	if !liked || count != 1 {
		t.Fatalf("1차: liked=%v count=%d, want true/1", liked, count)
	}
	if !svc.IsLiked(post.ID, fan.ID) {
		t.Fatal("IsLiked가 false — 좋아요 기록 안 됨")
	}

	// 2차 토글: 취소
	liked, count, err = svc.ToggleLike(post.ID, fan.ID)
	if err != nil {
		t.Fatalf("2차 토글 실패: %v", err)
	}
	if liked || count != 0 {
		t.Fatalf("2차: liked=%v count=%d, want false/0", liked, count)
	}
	if svc.IsLiked(post.ID, fan.ID) {
		t.Fatal("IsLiked가 true — 취소 반영 안 됨")
	}

	// 다른 유저의 좋아요는 독립적
	other := testutil.MakeUser(t, db, "other-fan")
	_, count, err = svc.ToggleLike(post.ID, other.ID)
	if err != nil {
		t.Fatalf("3차 토글 실패: %v", err)
	}
	if count != 1 {
		t.Fatalf("count=%d, want 1", count)
	}
}
