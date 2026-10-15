package sitemap

import (
	"testing"

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
	db.Delete(&gone)

	entries, err := NewRepository(db).GetSeriesEntries()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != kept.ID {
		t.Fatalf("entries = %+v, want only %d", entries, kept.ID)
	}
}
