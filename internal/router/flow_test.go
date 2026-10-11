package router

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"tolelom_api/internal/config"
	"tolelom_api/internal/model"
	"tolelom_api/internal/testutil"
	"tolelom_api/internal/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

const testJWTSecret = "flow-test-secret"

// newFlowApp wires the real router against an in-memory DB.
// Redis points at an unreachable port so the app runs without a cache.
func newFlowApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()
	db := testutil.SetupDB(t)
	cfg := &config.Config{
		JWTSecret:   testJWTSecret,
		Environment: "test",
		RedisAddr:   "127.0.0.1:1",
		UploadDir:   t.TempDir(),
		DB:          db,
	}
	app := fiber.New()
	cleanup := Setup(app, cfg)
	t.Cleanup(cleanup)
	return app, db
}

func accessToken(t *testing.T, u *model.User) string {
	t.Helper()
	access, _, err := utils.GenerateTokenPair(u, testJWTSecret)
	if err != nil {
		t.Fatalf("토큰 생성 실패: %v", err)
	}
	return access
}

func doRequest(t *testing.T, app *fiber.App, method, path, token, body string) int {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s 요청 실패: %v", method, path, err)
	}
	return resp.StatusCode
}

func TestFlow_PostCreatedAsPrivate_IsHiddenFromAnonymous(t *testing.T) {
	app, db := newFlowApp(t)
	owner := testutil.MakeUser(t, db, "owner")
	token := accessToken(t, owner)

	status := doRequest(t, app, http.MethodPost, "/api/v1/posts", token,
		`{"title":"비공개 글","content":"본문","is_public":false}`)
	if status != http.StatusCreated {
		t.Fatalf("글 생성 status = %d, want 201", status)
	}
	var created model.Post
	if err := db.Last(&created).Error; err != nil {
		t.Fatalf("생성된 글 조회 실패: %v", err)
	}

	if got := doRequest(t, app, http.MethodGet, "/api/v1/posts/"+strconv.FormatUint(uint64(created.ID), 10), "", ""); got != http.StatusForbidden {
		t.Errorf("익명 조회 status = %d, want 403", got)
	}
	if got := doRequest(t, app, http.MethodGet, "/api/v1/posts/"+strconv.FormatUint(uint64(created.ID), 10), token, ""); got != http.StatusOK {
		t.Errorf("작성자 조회 status = %d, want 200", got)
	}
}

// 운영에 이미 is_public=true 로 저장된 draft 가 있다. 공개 플래그와 무관하게 작성자만 봐야 한다.
func TestFlow_DraftFlaggedPublic_IsHiddenFromOthers(t *testing.T) {
	app, db := newFlowApp(t)
	owner := testutil.MakeUser(t, db, "owner")
	other := testutil.MakeUser(t, db, "other")
	draft := testutil.MakePost(t, db, owner.ID, testutil.WithStatus("draft"))
	path := "/api/v1/posts/" + strconv.FormatUint(uint64(draft.ID), 10)

	if got := doRequest(t, app, http.MethodGet, path, "", ""); got != http.StatusForbidden {
		t.Errorf("익명 조회 status = %d, want 403", got)
	}
	if got := doRequest(t, app, http.MethodGet, path, accessToken(t, other), ""); got != http.StatusForbidden {
		t.Errorf("타인 조회 status = %d, want 403", got)
	}
	if got := doRequest(t, app, http.MethodGet, path, accessToken(t, owner), ""); got != http.StatusOK {
		t.Errorf("작성자 조회 status = %d, want 200", got)
	}
}
