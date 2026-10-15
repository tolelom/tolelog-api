package series

import (
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"tolelom_api/internal/dto"
	"tolelom_api/internal/model"
	"tolelom_api/internal/testutil"
)

func makeSeries(t *testing.T, db *gorm.DB, userID uint, title string) *model.Series {
	t.Helper()
	s := &model.Series{Title: title, UserID: userID}
	if err := db.Create(s).Error; err != nil {
		t.Fatalf("series fixture 생성 실패: %v", err)
	}
	return s
}

func addPosts(t *testing.T, svc Service, seriesID, userID uint, posts ...*model.Post) {
	t.Helper()
	for i, p := range posts {
		if err := svc.AddPostToSeries(seriesID, p.ID, i+1, userID); err != nil {
			t.Fatalf("AddPostToSeries(%d): %v", p.ID, err)
		}
	}
}

func reloadPost(t *testing.T, db *gorm.DB, id uint) model.Post {
	t.Helper()
	var p model.Post
	if err := db.Unscoped().First(&p, id).Error; err != nil {
		t.Fatalf("post %d 다시 읽기 실패: %v", id, err)
	}
	return p
}

func strPtr(s string) *string { return &s }

func TestCreateSeries_LoadsAuthor(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := testutil.MakeUser(t, db, "owner")
	svc := NewService(db)

	s := &model.Series{Title: "Go 기초", UserID: owner.ID}
	if err := svc.CreateSeries(s); err != nil {
		t.Fatalf("CreateSeries: %v", err)
	}
	if s.ID == 0 {
		t.Fatal("ID 가 채워지지 않았다")
	}
	if s.User.Username != "owner" {
		t.Fatalf("User.Username = %q, want owner (응답에 작성자 이름이 나가야 한다)", s.User.Username)
	}
}

func TestGetSeriesByID(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := testutil.MakeUser(t, db, "owner")
	s := makeSeries(t, db, owner.ID, "C 언어 기초")
	other := makeSeries(t, db, owner.ID, "다른 시리즈")
	first := testutil.MakePost(t, db, owner.ID, testutil.WithTitle("#1"))
	second := testutil.MakePost(t, db, owner.ID, testutil.WithTitle("#2"))
	deleted := testutil.MakePost(t, db, owner.ID, testutil.WithTitle("삭제됨"))
	elsewhere := testutil.MakePost(t, db, owner.ID, testutil.WithTitle("다른 시리즈 글"))
	svc := NewService(db)

	addPosts(t, svc, s.ID, owner.ID, second, first, deleted)
	addPosts(t, svc, other.ID, owner.ID, elsewhere)
	if err := svc.ReorderPosts(s.ID, []uint{first.ID, second.ID, deleted.ID}, owner.ID); err != nil {
		t.Fatalf("ReorderPosts: %v", err)
	}
	if err := db.Delete(&model.Post{}, deleted.ID).Error; err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	t.Run("시리즈 순서대로 소속 글만 돌려준다", func(t *testing.T) {
		got, posts, err := svc.GetSeriesByID(s.ID)
		if err != nil {
			t.Fatalf("GetSeriesByID: %v", err)
		}
		if got.User.Username != "owner" {
			t.Fatalf("User.Username = %q, want owner", got.User.Username)
		}
		ids := make([]uint, len(posts))
		for i, p := range posts {
			ids[i] = p.ID
		}
		if want := []uint{first.ID, second.ID}; !equal(ids, want) {
			t.Fatalf("posts = %v, want %v (삭제된 글과 다른 시리즈 글은 빠져야 한다)", ids, want)
		}
	})

	t.Run("없는 시리즈", func(t *testing.T) {
		_, _, err := svc.GetSeriesByID(9999)
		if !errors.Is(err, ErrSeriesNotFound) {
			t.Fatalf("err = %v, want ErrSeriesNotFound", err)
		}
	})
}

func TestGetSeriesByUserID_CountsLivePostsPerSeries(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := testutil.MakeUser(t, db, "owner")
	stranger := testutil.MakeUser(t, db, "stranger")
	older := makeSeries(t, db, owner.ID, "먼저 만든 시리즈")
	newer := makeSeries(t, db, owner.ID, "나중 만든 시리즈")
	db.Model(older).UpdateColumn("created_at", older.CreatedAt.Add(-time.Hour))
	makeSeries(t, db, stranger.ID, "남의 시리즈")
	svc := NewService(db)

	kept := testutil.MakePost(t, db, owner.ID)
	removed := testutil.MakePost(t, db, owner.ID)
	addPosts(t, svc, older.ID, owner.ID, kept, removed)
	if err := db.Delete(&model.Post{}, removed.ID).Error; err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	list, counts, err := svc.GetSeriesByUserID(owner.ID)
	if err != nil {
		t.Fatalf("GetSeriesByUserID: %v", err)
	}
	if len(list) != 2 || list[0].ID != newer.ID || list[1].ID != older.ID {
		t.Fatalf("시리즈 목록 = %+v, want [newer, older] (본인 것만, 최신순)", list)
	}
	if counts[older.ID] != 1 {
		t.Fatalf("older 글 수 = %d, want 1 (삭제된 글은 세지 않는다)", counts[older.ID])
	}
	if counts[newer.ID] != 0 {
		t.Fatalf("newer 글 수 = %d, want 0", counts[newer.ID])
	}
}

func TestGetSeriesByUserID_NoSeries(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := testutil.MakeUser(t, db, "owner")

	list, counts, err := NewService(db).GetSeriesByUserID(owner.ID)
	if err != nil {
		t.Fatalf("GetSeriesByUserID: %v", err)
	}
	if len(list) != 0 || len(counts) != 0 {
		t.Fatalf("list=%v counts=%v, want 비어 있음", list, counts)
	}
}

func TestUpdateSeries(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := testutil.MakeUser(t, db, "owner")
	stranger := testutil.MakeUser(t, db, "stranger")
	svc := NewService(db)

	t.Run("보낸 필드만 바꾼다", func(t *testing.T) {
		s := makeSeries(t, db, owner.ID, "원래 제목")
		db.Model(s).UpdateColumn("description", "원래 설명")

		got, err := svc.UpdateSeries(s.ID, owner.ID, &dto.UpdateSeriesRequest{Description: strPtr("새 설명")})
		if err != nil {
			t.Fatalf("UpdateSeries: %v", err)
		}
		if got.Title != "원래 제목" || got.Description != "새 설명" {
			t.Fatalf("got = (%q, %q), want (원래 제목, 새 설명)", got.Title, got.Description)
		}
		if got.User.Username != "owner" {
			t.Fatalf("User.Username = %q, want owner", got.User.Username)
		}
	})

	t.Run("남의 시리즈는 거부하고 그대로 둔다", func(t *testing.T) {
		s := makeSeries(t, db, owner.ID, "원래 제목")

		_, err := svc.UpdateSeries(s.ID, stranger.ID, &dto.UpdateSeriesRequest{Title: strPtr("탈취")})
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("err = %v, want ErrUnauthorized", err)
		}
		var reloaded model.Series
		db.First(&reloaded, s.ID)
		if reloaded.Title != "원래 제목" {
			t.Fatalf("Title = %q, 바뀌면 안 된다", reloaded.Title)
		}
	})

	t.Run("없는 시리즈", func(t *testing.T) {
		_, err := svc.UpdateSeries(9999, owner.ID, &dto.UpdateSeriesRequest{Title: strPtr("x")})
		if !errors.Is(err, ErrSeriesNotFound) {
			t.Fatalf("err = %v, want ErrSeriesNotFound", err)
		}
	})
}

func TestDeleteSeries(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := testutil.MakeUser(t, db, "owner")
	stranger := testutil.MakeUser(t, db, "stranger")
	svc := NewService(db)

	t.Run("글은 남기고 시리즈 소속만 푼다", func(t *testing.T) {
		s := makeSeries(t, db, owner.ID, "지울 시리즈")
		p := testutil.MakePost(t, db, owner.ID)
		addPosts(t, svc, s.ID, owner.ID, p)

		if err := svc.DeleteSeries(s.ID, owner.ID); err != nil {
			t.Fatalf("DeleteSeries: %v", err)
		}
		if _, _, err := svc.GetSeriesByID(s.ID); !errors.Is(err, ErrSeriesNotFound) {
			t.Fatalf("삭제 뒤 조회 err = %v, want ErrSeriesNotFound", err)
		}
		got := reloadPost(t, db, p.ID)
		if got.DeletedAt.Valid {
			t.Fatal("시리즈를 지웠더니 글까지 지워졌다")
		}
		if got.SeriesID != nil || got.SeriesOrder != nil {
			t.Fatalf("series_id=%v series_order=%v, want 둘 다 nil", got.SeriesID, got.SeriesOrder)
		}
	})

	t.Run("남의 시리즈는 거부한다", func(t *testing.T) {
		s := makeSeries(t, db, owner.ID, "남이 못 지움")
		if err := svc.DeleteSeries(s.ID, stranger.ID); !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("err = %v, want ErrUnauthorized", err)
		}
		if _, _, err := svc.GetSeriesByID(s.ID); err != nil {
			t.Fatalf("거부됐는데 시리즈가 사라졌다: %v", err)
		}
	})

	t.Run("없는 시리즈", func(t *testing.T) {
		if err := svc.DeleteSeries(9999, owner.ID); !errors.Is(err, ErrSeriesNotFound) {
			t.Fatalf("err = %v, want ErrSeriesNotFound", err)
		}
	})
}

func TestAddPostToSeries_Rejections(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := testutil.MakeUser(t, db, "owner")
	stranger := testutil.MakeUser(t, db, "stranger")
	s := makeSeries(t, db, owner.ID, "시리즈")
	mine := testutil.MakePost(t, db, owner.ID)
	theirs := testutil.MakePost(t, db, stranger.ID)
	gone := testutil.MakePost(t, db, owner.ID)
	db.Delete(&model.Post{}, gone.ID)
	svc := NewService(db)

	cases := []struct {
		name     string
		seriesID uint
		postID   uint
		userID   uint
		want     error
	}{
		{"없는 시리즈", 9999, mine.ID, owner.ID, ErrSeriesNotFound},
		{"남의 시리즈", s.ID, theirs.ID, stranger.ID, ErrUnauthorized},
		{"남의 글", s.ID, theirs.ID, owner.ID, ErrPostNotOwned},
		{"삭제된 글", s.ID, gone.ID, owner.ID, ErrPostNotFound},
		{"없는 글", s.ID, 9999, owner.ID, ErrPostNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := svc.AddPostToSeries(c.seriesID, c.postID, 0, c.userID)
			if !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
		})
	}
	if got := reloadPost(t, db, theirs.ID); got.SeriesID != nil {
		t.Fatalf("거부된 글이 시리즈에 들어갔다: series_id=%d", *got.SeriesID)
	}
}

func TestAddPostToSeries_MovingFromAnotherSeriesGoesLast(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := testutil.MakeUser(t, db, "owner")
	from := makeSeries(t, db, owner.ID, "원래 시리즈")
	to := makeSeries(t, db, owner.ID, "옮길 시리즈")
	a := testutil.MakePost(t, db, owner.ID)
	b := testutil.MakePost(t, db, owner.ID)
	moving := testutil.MakePost(t, db, owner.ID)
	svc := NewService(db)
	addPosts(t, svc, to.ID, owner.ID, a, b)
	if err := svc.AddPostToSeries(from.ID, moving.ID, 1, owner.ID); err != nil {
		t.Fatalf("AddPostToSeries(from): %v", err)
	}

	if err := svc.AddPostToSeries(to.ID, moving.ID, 0, owner.ID); err != nil {
		t.Fatalf("AddPostToSeries(to): %v", err)
	}
	if got, want := seriesOrders(t, svc, to.ID), []uint{a.ID, b.ID, moving.ID}; !equal(got, want) {
		t.Fatalf("to = %v, want %v (원래 시리즈의 순서 1을 들고 오면 안 된다)", got, want)
	}
	if got := seriesOrders(t, svc, from.ID); len(got) != 0 {
		t.Fatalf("from = %v, want 비어 있음", got)
	}
}

func TestRemovePostFromSeries(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := testutil.MakeUser(t, db, "owner")
	stranger := testutil.MakeUser(t, db, "stranger")
	s := makeSeries(t, db, owner.ID, "시리즈")
	other := makeSeries(t, db, owner.ID, "다른 시리즈")
	inS := testutil.MakePost(t, db, owner.ID)
	inOther := testutil.MakePost(t, db, owner.ID)
	svc := NewService(db)
	addPosts(t, svc, s.ID, owner.ID, inS)
	addPosts(t, svc, other.ID, owner.ID, inOther)

	t.Run("남의 시리즈는 거부한다", func(t *testing.T) {
		if err := svc.RemovePostFromSeries(s.ID, inS.ID, stranger.ID); !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("err = %v, want ErrUnauthorized", err)
		}
		if got := seriesOrders(t, svc, s.ID); !equal(got, []uint{inS.ID}) {
			t.Fatalf("거부됐는데 글이 빠졌다: %v", got)
		}
	})

	t.Run("다른 시리즈의 글은 건드리지 않는다", func(t *testing.T) {
		if err := svc.RemovePostFromSeries(s.ID, inOther.ID, owner.ID); err != nil {
			t.Fatalf("RemovePostFromSeries: %v", err)
		}
		if got := seriesOrders(t, svc, other.ID); !equal(got, []uint{inOther.ID}) {
			t.Fatalf("other = %v, want [%d]", got, inOther.ID)
		}
	})

	t.Run("소속과 순서를 함께 지운다", func(t *testing.T) {
		if err := svc.RemovePostFromSeries(s.ID, inS.ID, owner.ID); err != nil {
			t.Fatalf("RemovePostFromSeries: %v", err)
		}
		got := reloadPost(t, db, inS.ID)
		if got.SeriesID != nil || got.SeriesOrder != nil {
			t.Fatalf("series_id=%v series_order=%v, want 둘 다 nil", got.SeriesID, got.SeriesOrder)
		}
	})

	t.Run("없는 시리즈", func(t *testing.T) {
		if err := svc.RemovePostFromSeries(9999, inS.ID, owner.ID); !errors.Is(err, ErrSeriesNotFound) {
			t.Fatalf("err = %v, want ErrSeriesNotFound", err)
		}
	})
}

func TestReorderPosts(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := testutil.MakeUser(t, db, "owner")
	stranger := testutil.MakeUser(t, db, "stranger")
	s := makeSeries(t, db, owner.ID, "시리즈")
	other := makeSeries(t, db, owner.ID, "다른 시리즈")
	a := testutil.MakePost(t, db, owner.ID)
	b := testutil.MakePost(t, db, owner.ID)
	c := testutil.MakePost(t, db, owner.ID)
	outsider := testutil.MakePost(t, db, owner.ID)
	outsiderNext := testutil.MakePost(t, db, owner.ID)
	svc := NewService(db)
	addPosts(t, svc, s.ID, owner.ID, a, b, c)
	addPosts(t, svc, other.ID, owner.ID, outsider, outsiderNext)

	t.Run("받은 순서대로 바꾼다", func(t *testing.T) {
		if err := svc.ReorderPosts(s.ID, []uint{c.ID, a.ID, b.ID}, owner.ID); err != nil {
			t.Fatalf("ReorderPosts: %v", err)
		}
		if got, want := seriesOrders(t, svc, s.ID), []uint{c.ID, a.ID, b.ID}; !equal(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("다른 시리즈 글 ID 가 섞여 와도 그 시리즈 순서를 건드리지 않는다", func(t *testing.T) {
		if err := svc.ReorderPosts(s.ID, []uint{c.ID, a.ID, b.ID, outsider.ID}, owner.ID); err != nil {
			t.Fatalf("ReorderPosts: %v", err)
		}
		if got := seriesOrders(t, svc, s.ID); len(got) != 3 {
			t.Fatalf("s = %v, 다른 시리즈 글이 섞이면 안 된다", got)
		}
		if got, want := seriesOrders(t, svc, other.ID), []uint{outsider.ID, outsiderNext.ID}; !equal(got, want) {
			t.Fatalf("other = %v, want %v", got, want)
		}
	})

	t.Run("남의 시리즈는 거부하고 순서를 지킨다", func(t *testing.T) {
		before := seriesOrders(t, svc, s.ID)
		if err := svc.ReorderPosts(s.ID, []uint{b.ID, a.ID, c.ID}, stranger.ID); !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("err = %v, want ErrUnauthorized", err)
		}
		if got := seriesOrders(t, svc, s.ID); !equal(got, before) {
			t.Fatalf("got %v, want %v", got, before)
		}
	})

	t.Run("없는 시리즈", func(t *testing.T) {
		if err := svc.ReorderPosts(9999, []uint{a.ID}, owner.ID); !errors.Is(err, ErrSeriesNotFound) {
			t.Fatalf("err = %v, want ErrSeriesNotFound", err)
		}
	})
}

func TestGetSeriesNavigation(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := testutil.MakeUser(t, db, "owner")
	s := makeSeries(t, db, owner.ID, "C 언어 기초")
	first := testutil.MakePost(t, db, owner.ID, testutil.WithTitle("#1"))
	middle := testutil.MakePost(t, db, owner.ID, testutil.WithTitle("#2"))
	last := testutil.MakePost(t, db, owner.ID, testutil.WithTitle("#3"))
	loose := testutil.MakePost(t, db, owner.ID)
	svc := NewService(db)
	addPosts(t, svc, s.ID, owner.ID, first, middle, last)

	navPostID := func(p *dto.SeriesNavPost) uint {
		if p == nil {
			return 0
		}
		return p.ID
	}
	cases := []struct {
		name               string
		post               *model.Post
		wantPrev, wantNext uint
		wantOrder          int
	}{
		{"첫 글은 이전 글이 없다", first, 0, middle.ID, 1},
		{"가운데 글은 앞뒤가 있다", middle, first.ID, last.ID, 2},
		{"마지막 글은 다음 글이 없다", last, middle.ID, 0, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			nav, err := svc.GetSeriesNavigation(c.post.ID)
			if err != nil || nav == nil {
				t.Fatalf("nav=%v err=%v", nav, err)
			}
			if nav.SeriesID != s.ID || nav.SeriesTitle != "C 언어 기초" || nav.TotalPosts != 3 {
				t.Fatalf("nav = %+v", nav)
			}
			if nav.CurrentOrder != c.wantOrder {
				t.Fatalf("CurrentOrder = %d, want %d", nav.CurrentOrder, c.wantOrder)
			}
			if got := navPostID(nav.PrevPost); got != c.wantPrev {
				t.Fatalf("PrevPost = %d, want %d", got, c.wantPrev)
			}
			if got := navPostID(nav.NextPost); got != c.wantNext {
				t.Fatalf("NextPost = %d, want %d", got, c.wantNext)
			}
		})
	}

	t.Run("이전·다음 글 제목을 싣는다", func(t *testing.T) {
		nav, _ := svc.GetSeriesNavigation(middle.ID)
		if nav.PrevPost.Title != "#1" || nav.NextPost.Title != "#3" {
			t.Fatalf("titles = (%q, %q), want (#1, #3)", nav.PrevPost.Title, nav.NextPost.Title)
		}
	})

	for name, id := range map[string]uint{"시리즈에 없는 글": loose.ID, "없는 글": 9999} {
		t.Run(name+"은 nil", func(t *testing.T) {
			nav, err := svc.GetSeriesNavigation(id)
			if err != nil || nav != nil {
				t.Fatalf("nav=%+v err=%v, want nil, nil", nav, err)
			}
		})
	}
}

func TestCountPostsInSeries_SkipsDeleted(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := testutil.MakeUser(t, db, "owner")
	s := makeSeries(t, db, owner.ID, "시리즈")
	kept := testutil.MakePost(t, db, owner.ID)
	removed := testutil.MakePost(t, db, owner.ID)
	svc := NewService(db)
	addPosts(t, svc, s.ID, owner.ID, kept, removed)
	db.Delete(&model.Post{}, removed.ID)

	got, err := svc.CountPostsInSeries(s.ID)
	if err != nil {
		t.Fatalf("CountPostsInSeries: %v", err)
	}
	if got != 1 {
		t.Fatalf("count = %d, want 1", got)
	}
}
