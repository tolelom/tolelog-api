package sitemap

import (
	"testing"

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
