package post

import (
	"errors"
	"slices"
	"sort"
	"testing"
	"time"

	"gorm.io/gorm"

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
	if liked, err := svc.IsLiked(post.ID, fan.ID); err != nil || !liked {
		t.Fatalf("IsLiked가 false — 좋아요 기록 안 됨 (err=%v)", err)
	}

	// 2차 토글: 취소
	liked, count, err = svc.ToggleLike(post.ID, fan.ID)
	if err != nil {
		t.Fatalf("2차 토글 실패: %v", err)
	}
	if liked || count != 0 {
		t.Fatalf("2차: liked=%v count=%d, want false/0", liked, count)
	}
	if liked, err := svc.IsLiked(post.ID, fan.ID); err != nil || liked {
		t.Fatalf("IsLiked가 true — 취소 반영 안 됨 (err=%v)", err)
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

// IsLiked는 조회 실패를 false로 삼키지 않고 오류로 돌려준다.
func TestIsLiked_DBError(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db, nil)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	if _, err := svc.IsLiked(1, 1); err == nil {
		t.Fatal("닫힌 DB에서 IsLiked가 오류를 내지 않았다")
	}
}

func tagNames(p *model.Post) []string {
	names := make([]string, len(p.Tags))
	for i, tag := range p.Tags {
		names[i] = tag.Name
	}
	sort.Strings(names)
	return names
}

func postIDs(posts []model.Post) []uint {
	ids := make([]uint, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
	}
	return ids
}

// makePostAt 은 created_at 을 정해 목록 정렬을 결정적으로 만든다.
func makePostAt(t *testing.T, db *gorm.DB, userID uint, minutesAgo int, opts ...testutil.PostOpt) *model.Post {
	t.Helper()
	p := testutil.MakePost(t, db, userID, opts...)
	at := time.Now().Add(-time.Duration(minutesAgo) * time.Minute)
	if err := db.Model(p).UpdateColumn("created_at", at).Error; err != nil {
		t.Fatalf("created_at 설정 실패: %v", err)
	}
	return p
}

func TestCreatePost(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db, nil)
	author := testutil.MakeUser(t, db, "author")

	t.Run("상태를 안 주면 발행 글이 되고 작성자를 싣는다", func(t *testing.T) {
		p := &model.Post{Title: "제목", Content: "본문", UserID: author.ID, IsPublic: true}
		if err := svc.CreatePost(p); err != nil {
			t.Fatalf("CreatePost: %v", err)
		}
		if p.Status != model.PostStatusPublished {
			t.Fatalf("Status = %q, want published", p.Status)
		}
		if p.User.Username != "author" {
			t.Fatalf("User.Username = %q, want author", p.User.Username)
		}
	})

	t.Run("태그 문자열을 다듬어 태그로 만든다", func(t *testing.T) {
		p := &model.Post{Title: "태그 글", Content: "본문", UserID: author.ID, TagsRaw: " go , ,rust,go"}
		if err := svc.CreatePost(p); err != nil {
			t.Fatalf("CreatePost: %v", err)
		}
		if got, want := tagNames(p), []string{"go", "rust"}; !slices.Equal(got, want) {
			t.Fatalf("tags = %v, want %v (공백·빈 값·중복은 빠져야 한다)", got, want)
		}
	})

	t.Run("이미 있는 태그는 다시 만들지 않는다", func(t *testing.T) {
		p := &model.Post{Title: "또 태그 글", Content: "본문", UserID: author.ID, TagsRaw: "go"}
		if err := svc.CreatePost(p); err != nil {
			t.Fatalf("CreatePost: %v", err)
		}
		var count int64
		db.Model(&model.Tag{}).Where("name = ?", "go").Count(&count)
		if count != 1 {
			t.Fatalf("go 태그 행 = %d, want 1", count)
		}
	})
}

func TestUpdatePost_Fields(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db, nil)
	owner := testutil.MakeUser(t, db, "owner")
	p := &model.Post{Title: "제목", Content: "본문", UserID: owner.ID, IsPublic: true, TagsRaw: "go,rust"}
	if err := svc.CreatePost(p); err != nil {
		t.Fatalf("CreatePost: %v", err)
	}

	t.Run("태그를 바꾸면 새 목록으로 갈아 끼운다", func(t *testing.T) {
		got, err := svc.UpdatePost(p.ID, owner.ID, &dto.UpdatePostRequest{Tags: strPtr("rust, c")})
		if err != nil {
			t.Fatalf("UpdatePost: %v", err)
		}
		if names, want := tagNames(got), []string{"c", "rust"}; !slices.Equal(names, want) {
			t.Fatalf("tags = %v, want %v", names, want)
		}
		if got.Title != "제목" {
			t.Fatalf("Title = %q, 보내지 않은 필드는 그대로여야 한다", got.Title)
		}
	})

	t.Run("빈 태그 문자열은 태그를 모두 뗀다", func(t *testing.T) {
		got, err := svc.UpdatePost(p.ID, owner.ID, &dto.UpdatePostRequest{Tags: strPtr("")})
		if err != nil {
			t.Fatalf("UpdatePost: %v", err)
		}
		if len(got.Tags) != 0 {
			t.Fatalf("tags = %v, want 없음", tagNames(got))
		}
	})

	t.Run("비공개로 돌리고 초안으로 내린다", func(t *testing.T) {
		private := false
		draft := string(model.PostStatusDraft)
		got, err := svc.UpdatePost(p.ID, owner.ID, &dto.UpdatePostRequest{IsPublic: &private, Status: &draft})
		if err != nil {
			t.Fatalf("UpdatePost: %v", err)
		}
		if got.IsPublic || got.Status != model.PostStatusDraft {
			t.Fatalf("IsPublic=%v Status=%q, want false/draft", got.IsPublic, got.Status)
		}
		if _, err := svc.GetPostByID(p.ID, nil); !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("익명 조회 err = %v, want ErrUnauthorized", err)
		}
	})

	t.Run("바꿀 필드가 없으면 거부한다", func(t *testing.T) {
		if _, err := svc.UpdatePost(p.ID, owner.ID, &dto.UpdatePostRequest{}); !errors.Is(err, ErrNoFieldsToUpdate) {
			t.Fatalf("err = %v, want ErrNoFieldsToUpdate", err)
		}
	})

	t.Run("없는 글", func(t *testing.T) {
		if _, err := svc.UpdatePost(9999, owner.ID, &dto.UpdatePostRequest{Title: strPtr("x")}); !errors.Is(err, ErrPostNotFound) {
			t.Fatalf("err = %v, want ErrPostNotFound", err)
		}
	})
}

func TestGetPostByID_ViewsAndMissing(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db, nil)
	owner := testutil.MakeUser(t, db, "owner")
	stranger := testutil.MakeUser(t, db, "stranger")
	p := testutil.MakePost(t, db, owner.ID)
	draft := testutil.MakePost(t, db, owner.ID, testutil.WithStatus("draft"))

	t.Run("조회할 때마다 조회수가 오른다", func(t *testing.T) {
		first, err := svc.GetPostByID(p.ID, nil)
		if err != nil {
			t.Fatalf("GetPostByID: %v", err)
		}
		second, _ := svc.GetPostByID(p.ID, nil)
		if first.ViewCount != 1 || second.ViewCount != 2 {
			t.Fatalf("view counts = %d, %d, want 1, 2", first.ViewCount, second.ViewCount)
		}
	})

	t.Run("거부된 조회는 조회수를 올리지 않는다", func(t *testing.T) {
		if _, err := svc.GetPostByID(draft.ID, &stranger.ID); !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("err = %v, want ErrUnauthorized (초안은 작성자만)", err)
		}
		var reloaded model.Post
		db.First(&reloaded, draft.ID)
		if reloaded.ViewCount != 0 {
			t.Fatalf("ViewCount = %d, want 0", reloaded.ViewCount)
		}
	})

	t.Run("없는 글과 삭제된 글", func(t *testing.T) {
		gone := testutil.MakePost(t, db, owner.ID)
		db.Delete(&model.Post{}, gone.ID)
		for _, id := range []uint{9999, gone.ID} {
			if _, err := svc.GetPostByID(id, &owner.ID); !errors.Is(err, ErrPostNotFound) {
				t.Fatalf("id %d err = %v, want ErrPostNotFound", id, err)
			}
		}
	})
}

func TestGetPublicPosts(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db, nil)
	author := testutil.MakeUser(t, db, "author")
	oldest := makePostAt(t, db, author.ID, 30)
	middle := makePostAt(t, db, author.ID, 20)
	newest := makePostAt(t, db, author.ID, 10)
	makePostAt(t, db, author.ID, 5, testutil.WithPrivate())
	makePostAt(t, db, author.ID, 5, testutil.WithStatus("draft"))
	gone := makePostAt(t, db, author.ID, 5)
	db.Delete(&model.Post{}, gone.ID)
	if _, err := svc.UpdatePost(middle.ID, author.ID, &dto.UpdatePostRequest{Tags: strPtr("go")}); err != nil {
		t.Fatalf("태그 달기: %v", err)
	}

	t.Run("공개 발행 글만 최신순으로", func(t *testing.T) {
		posts, total, err := svc.GetPublicPosts(1, 10, "")
		if err != nil {
			t.Fatalf("GetPublicPosts: %v", err)
		}
		if want := []uint{newest.ID, middle.ID, oldest.ID}; total != 3 || !slices.Equal(postIDs(posts), want) {
			t.Fatalf("total=%d posts=%v, want 3 %v", total, postIDs(posts), want)
		}
		if posts[0].User.Username != "author" {
			t.Fatalf("목록에 작성자가 실리지 않았다")
		}
	})

	t.Run("페이지를 나눠도 total 은 전체 수", func(t *testing.T) {
		posts, total, err := svc.GetPublicPosts(2, 2, "")
		if err != nil {
			t.Fatalf("GetPublicPosts: %v", err)
		}
		if total != 3 || !slices.Equal(postIDs(posts), []uint{oldest.ID}) {
			t.Fatalf("total=%d posts=%v, want 3 [%d]", total, postIDs(posts), oldest.ID)
		}
	})

	t.Run("태그로 거른다", func(t *testing.T) {
		posts, total, err := svc.GetPublicPosts(1, 10, " go ")
		if err != nil {
			t.Fatalf("GetPublicPosts: %v", err)
		}
		if total != 1 || !slices.Equal(postIDs(posts), []uint{middle.ID}) {
			t.Fatalf("total=%d posts=%v, want 1 [%d]", total, postIDs(posts), middle.ID)
		}
	})

	t.Run("잘못된 태그는 거부한다", func(t *testing.T) {
		if _, _, err := svc.GetPublicPosts(1, 10, "go;drop"); !errors.Is(err, ErrInvalidTag) {
			t.Fatalf("err = %v, want ErrInvalidTag", err)
		}
	})
}

func TestSearchPosts(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db, nil)
	author := testutil.MakeUser(t, db, "author")
	byTitle := makePostAt(t, db, author.ID, 20, testutil.WithTitle("가비지 컬렉션"))
	byContent := makePostAt(t, db, author.ID, 10, testutil.WithTitle("메모리"))
	db.Model(byContent).UpdateColumn("content", "Mark and Sweep 가비지 수집기")
	makePostAt(t, db, author.ID, 5, testutil.WithTitle("가비지 초안"), testutil.WithStatus("draft"))
	makePostAt(t, db, author.ID, 5, testutil.WithTitle("가비지 비공개"), testutil.WithPrivate())
	makePostAt(t, db, author.ID, 5, testutil.WithTitle("상관없는 글"))

	posts, total, err := svc.SearchPosts("가비지", 1, 10)
	if err != nil {
		t.Fatalf("SearchPosts: %v", err)
	}
	if want := []uint{byContent.ID, byTitle.ID}; total != 2 || !slices.Equal(postIDs(posts), want) {
		t.Fatalf("total=%d posts=%v, want 2 %v (제목·본문 일치, 숨은 글 제외, 최신순)", total, postIDs(posts), want)
	}

	posts, total, _ = svc.SearchPosts("가비지", 2, 1)
	if total != 2 || !slices.Equal(postIDs(posts), []uint{byTitle.ID}) {
		t.Fatalf("2쪽: total=%d posts=%v, want 2 [%d]", total, postIDs(posts), byTitle.ID)
	}
}
