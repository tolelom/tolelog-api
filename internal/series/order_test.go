package series

import (
	"testing"

	"tolelom_api/internal/model"
	"tolelom_api/internal/testutil"
)

func seriesOrders(t *testing.T, svc Service, seriesID uint) []uint {
	t.Helper()
	_, posts, err := svc.GetSeriesByID(seriesID)
	if err != nil {
		t.Fatalf("GetSeries: %v", err)
	}
	ids := make([]uint, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
	}
	return ids
}

func TestAddPostToSeries_WithoutOrderAppendsAndKeepsPlaceOnResave(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := testutil.MakeUser(t, db, "owner")
	s := model.Series{Title: "C 언어 기초", UserID: owner.ID}
	if err := db.Create(&s).Error; err != nil {
		t.Fatalf("create series: %v", err)
	}
	first := testutil.MakePost(t, db, owner.ID, testutil.WithTitle("#1"))
	second := testutil.MakePost(t, db, owner.ID, testutil.WithTitle("#2"))
	third := testutil.MakePost(t, db, owner.ID, testutil.WithTitle("#3"))
	svc := NewService(db)

	for i, p := range []*model.Post{first, second} {
		if err := svc.AddPostToSeries(s.ID, p.ID, i+1, owner.ID); err != nil {
			t.Fatalf("add %d: %v", p.ID, err)
		}
	}
	if err := svc.AddPostToSeries(s.ID, third.ID, 0, owner.ID); err != nil {
		t.Fatalf("add %d: %v", third.ID, err)
	}
	want := []uint{first.ID, second.ID, third.ID}
	if got := seriesOrders(t, svc, s.ID); !equal(got, want) {
		t.Fatalf("after adding: %v, want %v (the editor sends order 0 and the new post must go last)", got, want)
	}

	if err := svc.AddPostToSeries(s.ID, second.ID, 0, owner.ID); err != nil {
		t.Fatalf("resave: %v", err)
	}
	if got := seriesOrders(t, svc, s.ID); !equal(got, want) {
		t.Fatalf("after re-saving #2: %v, want %v (editing a post must not move it)", got, want)
	}
}

func equal(a, b []uint) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
