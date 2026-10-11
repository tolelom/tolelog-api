package router

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"tolelom_api/internal/model"
	"tolelom_api/internal/testutil"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

const secretTitle = "SECRET-TITLE"

func get(t *testing.T, app *fiber.App, path, token string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("GET %s 실패: %v", path, err)
	}
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func id(n uint) string { return strconv.FormatUint(uint64(n), 10) }

func TestFlow_CommentsOfHiddenPost_OnlyOwnerCanRead(t *testing.T) {
	app, db := newFlowApp(t)
	owner := testutil.MakeUser(t, db, "owner")
	other := testutil.MakeUser(t, db, "other")
	private := testutil.MakePost(t, db, owner.ID, testutil.WithPrivate())
	draft := testutil.MakePost(t, db, owner.ID, testutil.WithStatus("draft"))

	for name, p := range map[string]*model.Post{"private": private, "draft": draft} {
		testutil.MakeComment(t, db, p.ID, owner.ID, "비밀 댓글")
		path := "/api/v1/posts/" + id(p.ID) + "/comments"

		if status, _ := get(t, app, path, ""); status != http.StatusForbidden {
			t.Errorf("%s: 익명 조회 status = %d, want 403", name, status)
		}
		if status, _ := get(t, app, path, accessToken(t, other)); status != http.StatusForbidden {
			t.Errorf("%s: 타인 조회 status = %d, want 403", name, status)
		}
		if status, body := get(t, app, path, accessToken(t, owner)); status != http.StatusOK || !strings.Contains(body, "비밀 댓글") {
			t.Errorf("%s: 작성자 조회 status = %d, body 에 댓글 포함 = %v", name, status, strings.Contains(body, "비밀 댓글"))
		}
	}
}

// seriesWithHiddenPost: 공개 글(1번) 다음에 비공개 글(2번)이 이어지는 시리즈
func seriesWithHiddenPost(t *testing.T, db *gorm.DB, owner *model.User) (series *model.Series, public, hidden *model.Post) {
	t.Helper()
	series = &model.Series{Title: "시리즈", UserID: owner.ID}
	if err := db.Create(series).Error; err != nil {
		t.Fatalf("시리즈 생성 실패: %v", err)
	}
	public = testutil.MakePost(t, db, owner.ID, testutil.WithTitle("공개 글"))
	hidden = testutil.MakePost(t, db, owner.ID, testutil.WithTitle(secretTitle), testutil.WithPrivate())
	db.Model(public).Updates(map[string]interface{}{"series_id": series.ID, "series_order": 1})
	db.Model(hidden).Updates(map[string]interface{}{"series_id": series.ID, "series_order": 2})
	return series, public, hidden
}

func TestFlow_SeriesDetail_HidesPrivatePostsFromOthers(t *testing.T) {
	app, db := newFlowApp(t)
	owner := testutil.MakeUser(t, db, "owner")
	series, _, _ := seriesWithHiddenPost(t, db, owner)
	path := "/api/v1/series/" + id(series.ID)

	if _, body := get(t, app, path, ""); strings.Contains(body, secretTitle) {
		t.Errorf("익명 시리즈 조회에 비공개 글 제목이 노출됨")
	}
	if _, body := get(t, app, path, accessToken(t, owner)); !strings.Contains(body, secretTitle) {
		t.Errorf("작성자 시리즈 조회에 자기 비공개 글이 없음")
	}
}

func TestFlow_SeriesNav_HidesPrivatePostsFromOthers(t *testing.T) {
	app, db := newFlowApp(t)
	owner := testutil.MakeUser(t, db, "owner")
	_, public, hidden := seriesWithHiddenPost(t, db, owner)

	if _, body := get(t, app, "/api/v1/posts/"+id(public.ID)+"/series-nav", ""); strings.Contains(body, secretTitle) {
		t.Errorf("익명 series-nav 의 다음 글로 비공개 글 제목이 노출됨")
	}
	if _, body := get(t, app, "/api/v1/posts/"+id(hidden.ID)+"/series-nav", ""); strings.Contains(body, "시리즈") {
		t.Errorf("익명에게 비공개 글의 series-nav 가 응답됨: %s", body)
	}
	if _, body := get(t, app, "/api/v1/posts/"+id(public.ID)+"/series-nav", accessToken(t, owner)); !strings.Contains(body, secretTitle) {
		t.Errorf("작성자 series-nav 에 자기 비공개 글이 없음")
	}
}
