package tag

import (
	"testing"

	"gorm.io/gorm"

	"tolelom_api/internal/model"
	"tolelom_api/internal/testutil"
)

func tagPost(t *testing.T, db *gorm.DB, p *model.Post, names ...string) {
	t.Helper()
	var tags []model.Tag
	for _, name := range names {
		var tag model.Tag
		if err := db.Where("name = ?", name).FirstOrCreate(&tag, model.Tag{Name: name}).Error; err != nil {
			t.Fatalf("tag %q: %v", name, err)
		}
		tags = append(tags, tag)
	}
	if err := db.Model(p).Association("Tags").Replace(tags); err != nil {
		t.Fatalf("태그 달기: %v", err)
	}
}

func TestGetTags_CountsOnlyPublishedPublicPosts(t *testing.T) {
	db := testutil.SetupDB(t)
	author := testutil.MakeUser(t, db, "author")
	tagPost(t, db, testutil.MakePost(t, db, author.ID), "go", "rust")
	tagPost(t, db, testutil.MakePost(t, db, author.ID), "go")
	tagPost(t, db, testutil.MakePost(t, db, author.ID), "c")
	tagPost(t, db, testutil.MakePost(t, db, author.ID, testutil.WithPrivate()), "go", "비공개만")
	tagPost(t, db, testutil.MakePost(t, db, author.ID, testutil.WithStatus("draft")), "go", "초안만")
	gone := testutil.MakePost(t, db, author.ID)
	tagPost(t, db, gone, "go", "삭제만")
	db.Delete(&model.Post{}, gone.ID)
	svc := NewService(db)

	summary := func(sort string, limit int) map[string]int64 {
		t.Helper()
		res, err := svc.GetTags(sort, limit)
		if err != nil {
			t.Fatalf("GetTags: %v", err)
		}
		if res.Total != int64(len(res.Tags)) {
			t.Fatalf("Total = %d, len = %d", res.Total, len(res.Tags))
		}
		out := map[string]int64{}
		for _, tag := range res.Tags {
			out[tag.Name] = tag.PostCount
		}
		return out
	}

	t.Run("숨은 글의 태그와 글 수는 드러나지 않는다", func(t *testing.T) {
		got := summary("popular", 50)
		want := map[string]int64{"go": 2, "rust": 1, "c": 1}
		if len(got) != len(want) {
			t.Fatalf("tags = %v, want %v", got, want)
		}
		for name, count := range want {
			if got[name] != count {
				t.Fatalf("tags = %v, want %v", got, want)
			}
		}
	})

	t.Run("인기순은 글 수가 많은 태그부터", func(t *testing.T) {
		res, _ := svc.GetTags("popular", 1)
		if len(res.Tags) != 1 || res.Tags[0].Name != "go" {
			t.Fatalf("tags = %+v, want [go]", res.Tags)
		}
	})

	t.Run("이름순", func(t *testing.T) {
		res, _ := svc.GetTags("name", 50)
		var names []string
		for _, tag := range res.Tags {
			names = append(names, tag.Name)
		}
		if len(names) != 3 || names[0] != "c" || names[1] != "go" || names[2] != "rust" {
			t.Fatalf("names = %v, want [c go rust]", names)
		}
	})
}
