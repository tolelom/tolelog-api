package sitemap

import (
	"testing"

	"gorm.io/gorm"

	"tolelom_api/internal/model"
	"tolelom_api/internal/testutil"
)

func TestGetPublicAuthorEntries_OnlyAuthorsWithPublishedPublicPosts(t *testing.T) {
	db := testutil.SetupDB(t)
	writer := testutil.MakeUser(t, db, "writer")
	privateOnly := testutil.MakeUser(t, db, "private_only")
	draftOnly := testutil.MakeUser(t, db, "draft_only")
	testutil.MakePost(t, db, writer.ID)
	testutil.MakePost(t, db, writer.ID, testutil.WithTitle("second"))
	testutil.MakePost(t, db, privateOnly.ID, testutil.WithPrivate())
	testutil.MakePost(t, db, draftOnly.ID, testutil.WithStatus("draft"))

	entries, err := NewRepository(db).GetPublicAuthorEntries()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != writer.ID {
		t.Fatalf("entries = %+v, want only writer %d", entries, writer.ID)
	}
}

func TestGetPublicPostEntries_OnlyPublishedPublicPosts(t *testing.T) {
	db := testutil.SetupDB(t)
	writer := testutil.MakeUser(t, db, "writer")
	public := testutil.MakePost(t, db, writer.ID, testutil.WithTitle("공개 글"))
	testutil.MakePost(t, db, writer.ID, testutil.WithPrivate())
	testutil.MakePost(t, db, writer.ID, testutil.WithStatus("draft"))
	gone := testutil.MakePost(t, db, writer.ID)
	db.Delete(&model.Post{}, gone.ID)

	entries, err := NewRepository(db).GetPublicPostEntries()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != public.ID || entries[0].Title != "공개 글" {
		t.Fatalf("entries = %+v, want only %d", entries, public.ID)
	}
}

func TestGetSeriesEntries_SkipsDeletedSeries(t *testing.T) {
	db := testutil.SetupDB(t)
	writer := testutil.MakeUser(t, db, "writer")
	kept := model.Series{Title: "남은 시리즈", UserID: writer.ID}
	gone := model.Series{Title: "지운 시리즈", UserID: writer.ID}
	db.Create(&kept)
	db.Create(&gone)
	inSeries(t, db, testutil.MakePost(t, db, writer.ID), kept.ID)
	inSeries(t, db, testutil.MakePost(t, db, writer.ID), gone.ID)
	db.Delete(&gone)

	entries, err := NewRepository(db).GetSeriesEntries()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != kept.ID {
		t.Fatalf("entries = %+v, want only %d", entries, kept.ID)
	}
}

func inSeries(t *testing.T, db *gorm.DB, p *model.Post, seriesID uint) {
	t.Helper()
	if err := db.Model(&model.Post{}).Where("id = ?", p.ID).Update("series_id", seriesID).Error; err != nil {
		t.Fatal(err)
	}
}

// 시리즈 페이지는 공개 글만 보여 주므로, 공개 발행 글이 하나도 없는 시리즈는 sitemap 에 넣지 않는다.
func TestGetSeriesEntries_OnlySeriesWithPublicPosts(t *testing.T) {
	db := testutil.SetupDB(t)
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatal(err)
	}
	writer := testutil.MakeUser(t, db, "writer")
	mk := func(title string) *model.Series {
		s := &model.Series{Title: title, UserID: writer.ID}
		if err := db.Create(s).Error; err != nil {
			t.Fatal(err)
		}
		return s
	}
	public := mk("공개 글 있음")
	draftOnly := mk("초안만")
	privateOnly := mk("비공개만")
	deletedOnly := mk("지운 공개 글만")
	mk("빈 시리즈")

	inSeries(t, db, testutil.MakePost(t, db, writer.ID), public.ID)
	inSeries(t, db, testutil.MakePost(t, db, writer.ID, testutil.WithPrivate()), public.ID)
	inSeries(t, db, testutil.MakePost(t, db, writer.ID, testutil.WithStatus("draft")), draftOnly.ID)
	inSeries(t, db, testutil.MakePost(t, db, writer.ID, testutil.WithPrivate()), privateOnly.ID)
	gone := testutil.MakePost(t, db, writer.ID)
	inSeries(t, db, gone, deletedOnly.ID)
	db.Delete(&model.Post{}, gone.ID)

	entries, err := NewRepository(db).GetSeriesEntries()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != public.ID {
		t.Fatalf("entries = %+v, want only series %d", entries, public.ID)
	}
}
