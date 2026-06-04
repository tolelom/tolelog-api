# Backend Critical Path Tests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 서비스 층 권한/보안 크리티컬 경로 12개를 SQLite in-memory DB 기반으로 테스트한다.

**Architecture:** `internal/testutil/` 공유 패키지(SetupDB + fixture factory)를 신설하고, 각 도메인에 `service_db_test.go`를 추가해 실제 GORM 쿼리 + 권한 분기를 검증한다. Redis는 `nil` 주입(서비스가 nil 가드 보유).

**Tech Stack:** Go 1.25, GORM, `github.com/glebarez/sqlite` (pure-Go, CGO 불필요), 표준 `go test`

**스펙:** `docs/superpowers/specs/2026-06-04-backend-critical-path-tests-design.md`

**주의 — 기존 구현이 이미 존재하므로** 새 테스트는 원칙적으로 즉시 PASS 해야 한다.
FAIL 이 나면 (a) 테스트 코드 결함 또는 (b) **실제 보안 버그 발견**이다. (b)면 수정 전에 보고할 것.

**작업 디렉터리:** `fiber_api_server/` (모든 명령은 이 디렉터리에서 실행)

---

### Task 1: SQLite 의존성 + testutil.SetupDB

**Files:**
- Modify: `go.mod` (go get으로 자동)
- Create: `internal/testutil/db.go`
- Test: `internal/testutil/db_test.go`

- [ ] **Step 1: 의존성 추가**

Run: `go get github.com/glebarez/sqlite`
Expected: go.mod/go.sum에 `github.com/glebarez/sqlite` 추가됨

- [ ] **Step 2: 실패하는 스모크 테스트 작성**

`internal/testutil/db_test.go`:

```go
package testutil

import (
	"testing"

	"tolelom_api/internal/model"
)

func TestSetupDB_MigratesAllModels(t *testing.T) {
	db := SetupDB(t)

	// 전 모델 테이블이 존재하고 INSERT 가능해야 한다
	u := &model.User{Username: "smoke", PasswordHash: "x"}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("user insert 실패: %v", err)
	}
	p := &model.Post{Title: "t", Content: "c", UserID: u.ID}
	if err := db.Create(p).Error; err != nil {
		t.Fatalf("post insert 실패: %v", err)
	}
	c := &model.Comment{PostID: p.ID, UserID: u.ID, Content: "hi"}
	if err := db.Create(c).Error; err != nil {
		t.Fatalf("comment insert 실패: %v", err)
	}
	l := &model.PostLike{PostID: p.ID, UserID: u.ID}
	if err := db.Create(l).Error; err != nil {
		t.Fatalf("post_like insert 실패: %v", err)
	}
}

func TestSetupDB_IsolatedBetweenCalls(t *testing.T) {
	db1 := SetupDB(t)
	db2 := SetupDB(t)

	db1.Create(&model.User{Username: "only-in-db1", PasswordHash: "x"})

	var count int64
	db2.Model(&model.User{}).Count(&count)
	if count != 0 {
		t.Fatalf("DB 격리 실패: db2에 %d명 존재", count)
	}
}
```

- [ ] **Step 3: 실패 확인**

Run: `go test ./internal/testutil/ -v`
Expected: COMPILE FAIL — `undefined: SetupDB`

- [ ] **Step 4: SetupDB 구현**

`internal/testutil/db.go`:

```go
// Package testutil provides shared test helpers: in-memory DB setup and fixtures.
package testutil

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"tolelom_api/internal/model"
)

// SetupDB returns a fresh in-memory SQLite DB with all models migrated.
// 각 호출마다 독립된 DB — 테스트 간 격리 보장.
func SetupDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("in-memory sqlite 열기 실패: %v", err)
	}

	// :memory: 는 커넥션마다 별도 DB가 되므로 커넥션을 1개로 고정해야 한다.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql.DB 핸들 획득 실패: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(
		&model.User{},
		&model.Tag{},
		&model.Series{},
		&model.Post{},
		&model.Comment{},
		&model.PostLike{},
	); err != nil {
		t.Fatalf("auto-migrate 실패: %v", err)
	}

	return db
}
```

- [ ] **Step 5: 통과 확인**

Run: `go test ./internal/testutil/ -v`
Expected: PASS (2 tests)

⚠️ AutoMigrate가 `type:longtext` 등 MySQL 타입 태그에서 실패하면 — SQLite는 타입명에 관대해 보통 통과하지만, 실패 시 에러 그대로 보고하고 멈출 것 (모델 수정은 스코프 외 결정 필요).

- [ ] **Step 6: 커밋**

```bash
git add go.mod go.sum internal/testutil/
git commit -m "test: testutil 패키지 추가 — SQLite in-memory DB 셋업"
```

---

### Task 2: Fixture Factory

**Files:**
- Create: `internal/testutil/fixtures.go`
- Test: `internal/testutil/fixtures_test.go`

- [ ] **Step 1: 실패하는 테스트 작성**

`internal/testutil/fixtures_test.go`:

```go
package testutil

import (
	"testing"

	"tolelom_api/internal/utils"
)

func TestMakeUser_CreatesUserWithValidBcryptHash(t *testing.T) {
	db := SetupDB(t)
	u := MakeUser(t, db, "alice")

	if u.ID == 0 {
		t.Fatal("ID가 채워지지 않음")
	}
	if u.Username != "alice" {
		t.Fatalf("username = %q, want alice", u.Username)
	}
	// DefaultPassword 로 검증 가능한 bcrypt 해시여야 한다 (Login 테스트에서 사용)
	if !utils.CheckPasswordHash(DefaultPassword, u.PasswordHash) {
		t.Fatal("PasswordHash가 DefaultPassword와 불일치")
	}
}

func TestMakePost_DefaultsAndOptions(t *testing.T) {
	db := SetupDB(t)
	u := MakeUser(t, db, "bob")

	pub := MakePost(t, db, u.ID)
	if !pub.IsPublic || pub.Status != "published" {
		t.Fatalf("기본값: IsPublic=%v Status=%q, want true/published", pub.IsPublic, pub.Status)
	}

	priv := MakePost(t, db, u.ID, WithPrivate())
	if priv.IsPublic {
		t.Fatal("WithPrivate 적용 안 됨")
	}

	draft := MakePost(t, db, u.ID, WithStatus("draft"))
	if draft.Status != "draft" {
		t.Fatalf("WithStatus 적용 안 됨: %q", draft.Status)
	}
}

func TestMakeComment_Creates(t *testing.T) {
	db := SetupDB(t)
	u := MakeUser(t, db, "carol")
	p := MakePost(t, db, u.ID)

	c := MakeComment(t, db, p.ID, u.ID, "댓글 내용")
	if c.ID == 0 || c.Content != "댓글 내용" {
		t.Fatalf("comment 생성 실패: %+v", c)
	}
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./internal/testutil/ -v`
Expected: COMPILE FAIL — `undefined: MakeUser` 등

- [ ] **Step 3: Fixture 구현**

`internal/testutil/fixtures.go`:

```go
package testutil

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"tolelom_api/internal/model"
	"tolelom_api/internal/utils"
)

// DefaultPassword is the plaintext password for all fixture users.
// 실제 비밀값 아님 — Login 테스트의 bcrypt 검증에 사용하는 테스트 전용 평문.
const DefaultPassword = "fixture-plain-pw-1!"

// MakeUser creates a user with a real bcrypt hash of DefaultPassword.
func MakeUser(t *testing.T, db *gorm.DB, username string) *model.User {
	t.Helper()
	hash, err := utils.HashPassword(DefaultPassword)
	if err != nil {
		t.Fatalf("비밀번호 해시 실패: %v", err)
	}
	u := &model.User{
		Username:     username,
		PasswordHash: hash,
		LastLogin:    time.Now(),
	}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("user fixture 생성 실패 (%s): %v", username, err)
	}
	return u
}

// PostOpt customizes a fixture post.
type PostOpt func(*model.Post)

// WithPrivate marks the post as private (is_public=false).
func WithPrivate() PostOpt {
	return func(p *model.Post) { p.IsPublic = false }
}

// WithStatus sets the post status (e.g. "draft").
func WithStatus(status string) PostOpt {
	return func(p *model.Post) { p.Status = status }
}

// WithTitle sets the post title.
func WithTitle(title string) PostOpt {
	return func(p *model.Post) { p.Title = title }
}

// MakePost creates a published public post by default.
func MakePost(t *testing.T, db *gorm.DB, userID uint, opts ...PostOpt) *model.Post {
	t.Helper()
	p := &model.Post{
		Title:    "테스트 글",
		Content:  "본문",
		UserID:   userID,
		IsPublic: true,
		Status:   "published",
	}
	for _, opt := range opts {
		opt(p)
	}
	if err := db.Create(p).Error; err != nil {
		t.Fatalf("post fixture 생성 실패: %v", err)
	}
	return p
}

// MakeComment creates a comment on a post.
func MakeComment(t *testing.T, db *gorm.DB, postID, userID uint, content string) *model.Comment {
	t.Helper()
	c := &model.Comment{PostID: postID, UserID: userID, Content: content}
	if err := db.Create(c).Error; err != nil {
		t.Fatalf("comment fixture 생성 실패: %v", err)
	}
	return c
}
```

- [ ] **Step 4: 통과 확인**

Run: `go test ./internal/testutil/ -v`
Expected: PASS (5 tests)

- [ ] **Step 5: 커밋**

```bash
git add internal/testutil/fixtures.go internal/testutil/fixtures_test.go
git commit -m "test: testutil fixture factory 추가 (MakeUser/MakePost/MakeComment)"
```

---

### Task 3: Post — GetPostByID 접근 제어 (경로 1·2)

**Files:**
- Create: `internal/post/service_db_test.go`

기존 `internal/post/service_test.go`(pure 함수 테스트)는 건드리지 않는다.

- [ ] **Step 1: 테스트 작성**

`internal/post/service_db_test.go`:

```go
package post

import (
	"errors"
	"testing"

	"tolelom_api/internal/testutil"
)

// 경로 1: is_public=false 글을 비로그인/타인이 조회 시 차단
// 경로 2: 본인은 자기 비공개 글 조회 가능
func TestGetPostByID_PrivatePostAccessControl(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db, nil) // Redis nil — 서비스가 nil 가드 보유

	owner := testutil.MakeUser(t, db, "owner")
	other := testutil.MakeUser(t, db, "other")
	private := testutil.MakePost(t, db, owner.ID, testutil.WithPrivate())

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
		public := testutil.MakePost(t, db, owner.ID)
		got, err := svc.GetPostByID(public.ID, nil)
		if err != nil {
			t.Fatalf("공개 글 조회 실패: %v", err)
		}
		if got.ID != public.ID {
			t.Fatalf("got.ID = %d, want %d", got.ID, public.ID)
		}
	})
}
```

- [ ] **Step 2: 통과 확인 (기존 구현 검증)**

Run: `go test ./internal/post/ -run TestGetPostByID_PrivatePostAccessControl -v`
Expected: PASS — FAIL이면 테스트 결함인지 실제 버그인지 분석 후, 버그면 보고하고 멈출 것

- [ ] **Step 3: 커밋**

```bash
git add internal/post/service_db_test.go
git commit -m "test: GetPostByID 비공개 글 접근 제어 테스트 (경로 1-2)"
```

---

### Task 4: Post — UpdatePost/DeletePost 소유권 (경로 3·4)

**Files:**
- Modify: `internal/post/service_db_test.go` (append)

- [ ] **Step 1: 테스트 추가**

`internal/post/service_db_test.go`에 append (import 블록에 `"tolelom_api/internal/dto"`, `"tolelom_api/internal/model"` 추가):

```go
func strPtr(s string) *string { return &s }

// 경로 3: 본인 글만 수정 가능
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
		db.Model(&model.Post{}).Where("id = ?", post.ID).Count(&count)
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
```

- [ ] **Step 2: 통과 확인**

Run: `go test ./internal/post/ -run 'TestUpdatePost_OwnershipEnforced|TestDeletePost_OwnershipEnforced' -v`
Expected: PASS

- [ ] **Step 3: 커밋**

```bash
git add internal/post/service_db_test.go
git commit -m "test: UpdatePost/DeletePost 소유권 검증 테스트 (경로 3-4)"
```

---

### Task 5: Post — GetUserPosts 가시성 / GetDrafts (경로 5·6)

**Files:**
- Modify: `internal/post/service_db_test.go` (append)

- [ ] **Step 1: 테스트 추가**

```go
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
		_, total, err := svc.GetUserPosts(owner.ID, &viewer.ID, 1, 10, "")
		if err != nil {
			t.Fatalf("조회 실패: %v", err)
		}
		if total != 1 {
			t.Fatalf("total=%d, want 1", total)
		}
	})

	t.Run("본인은 전부", func(t *testing.T) {
		_, total, err := svc.GetUserPosts(owner.ID, &owner.ID, 1, 10, "")
		if err != nil {
			t.Fatalf("조회 실패: %v", err)
		}
		if total != 3 {
			t.Fatalf("total=%d, want 3", total)
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
```

- [ ] **Step 2: 통과 확인**

Run: `go test ./internal/post/ -run 'TestGetUserPosts_VisibilityByViewer|TestGetDrafts_OnlyOwnDrafts' -v`
Expected: PASS

- [ ] **Step 3: 커밋**

```bash
git add internal/post/service_db_test.go
git commit -m "test: GetUserPosts 가시성·GetDrafts 소유 검증 테스트 (경로 5-6)"
```

---

### Task 6: Post — ToggleLike (경로 7)

**Files:**
- Modify: `internal/post/service_db_test.go` (append)

- [ ] **Step 1: 테스트 추가**

```go
// 경로 7: 같은 유저 두 번 토글 시 liked 상태/count 증감 정상
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
```

- [ ] **Step 2: 통과 확인**

Run: `go test ./internal/post/ -run TestToggleLike_TogglesAndCounts -v`
Expected: PASS

- [ ] **Step 3: 커밋**

```bash
git add internal/post/service_db_test.go
git commit -m "test: ToggleLike 토글/카운트 검증 테스트 (경로 7)"
```

---

### Task 7: Comment — 접근 제어 (경로 8·9·10)

**Files:**
- Create: `internal/comment/service_db_test.go`

⚠️ **설계 노트:** 현재 구현·Swagger 문서 모두 "댓글 작성자만 삭제 가능"이다 (글 작성자의 타인 댓글 삭제 권한 없음 — WIP 문서의 "or 글 작성자"는 미구현 가정이었음). 테스트는 **현재 동작(작성자만)** 을 인코딩한다.

- [ ] **Step 1: 테스트 작성**

`internal/comment/service_db_test.go`:

```go
package comment

import (
	"errors"
	"testing"

	"tolelom_api/internal/model"
	"tolelom_api/internal/testutil"
)

// 경로 8: 비공개 글에 타인 댓글 차단, 글 작성자 본인은 가능
func TestCreateComment_PrivatePostAccessControl(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db)

	owner := testutil.MakeUser(t, db, "owner")
	other := testutil.MakeUser(t, db, "other")
	private := testutil.MakePost(t, db, owner.ID, testutil.WithPrivate())

	t.Run("타인의 비공개 글 댓글 차단", func(t *testing.T) {
		c := &model.Comment{PostID: private.ID, UserID: other.ID, Content: "몰래 댓글"}
		err := svc.CreateComment(c)
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("err = %v, want ErrUnauthorized", err)
		}
	})

	t.Run("글 작성자 본인은 비공개 글에 댓글 가능", func(t *testing.T) {
		c := &model.Comment{PostID: private.ID, UserID: owner.ID, Content: "셀프 댓글"}
		if err := svc.CreateComment(c); err != nil {
			t.Fatalf("본인 댓글 실패: %v", err)
		}
		if c.ID == 0 {
			t.Fatal("comment ID 미할당")
		}
	})

	t.Run("공개 글에는 누구나 댓글 가능", func(t *testing.T) {
		public := testutil.MakePost(t, db, owner.ID)
		c := &model.Comment{PostID: public.ID, UserID: other.ID, Content: "공개 댓글"}
		if err := svc.CreateComment(c); err != nil {
			t.Fatalf("공개 글 댓글 실패: %v", err)
		}
	})
}

// 경로 9: 댓글 작성자 본인만 삭제 가능 (현재 구현 — 글 작성자 권한 없음)
func TestDeleteComment_AuthorOnly(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db)

	postOwner := testutil.MakeUser(t, db, "post-owner")
	commenter := testutil.MakeUser(t, db, "commenter")
	post := testutil.MakePost(t, db, postOwner.ID)
	comment := testutil.MakeComment(t, db, post.ID, commenter.ID, "지워질 댓글")

	t.Run("글 작성자도 타인 댓글 삭제 불가 (현재 구현)", func(t *testing.T) {
		err := svc.DeleteComment(comment.ID, postOwner.ID)
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("err = %v, want ErrUnauthorized", err)
		}
	})

	t.Run("작성자 본인은 삭제 가능 (soft delete)", func(t *testing.T) {
		if err := svc.DeleteComment(comment.ID, commenter.ID); err != nil {
			t.Fatalf("본인 삭제 실패: %v", err)
		}
		var count int64
		db.Model(&model.Comment{}).Where("id = ?", comment.ID).Count(&count)
		if count != 0 {
			t.Fatal("soft delete 후에도 기본 쿼리에 노출됨")
		}
	})
}

// 경로 10: 작성자 본인만 수정 가능
func TestUpdateComment_AuthorOnly(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db)

	commenter := testutil.MakeUser(t, db, "commenter")
	intruder := testutil.MakeUser(t, db, "intruder")
	post := testutil.MakePost(t, db, commenter.ID)
	comment := testutil.MakeComment(t, db, post.ID, commenter.ID, "원본")

	t.Run("타인 수정 차단", func(t *testing.T) {
		_, err := svc.UpdateComment(comment.ID, intruder.ID, "변조")
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("err = %v, want ErrUnauthorized", err)
		}
		var reloaded model.Comment
		if err := db.First(&reloaded, comment.ID).Error; err != nil {
			t.Fatalf("reload 실패: %v", err)
		}
		if reloaded.Content != "원본" {
			t.Fatalf("차단됐는데 DB 변경됨: %q", reloaded.Content)
		}
	})

	t.Run("본인 수정 가능 + IsEdited 마킹", func(t *testing.T) {
		updated, err := svc.UpdateComment(comment.ID, commenter.ID, "수정본")
		if err != nil {
			t.Fatalf("본인 수정 실패: %v", err)
		}
		if updated.Content != "수정본" || !updated.IsEdited {
			t.Fatalf("content=%q isEdited=%v, want 수정본/true", updated.Content, updated.IsEdited)
		}
	})
}
```

- [ ] **Step 2: 통과 확인**

Run: `go test ./internal/comment/ -run 'TestCreateComment_PrivatePostAccessControl|TestDeleteComment_AuthorOnly|TestUpdateComment_AuthorOnly' -v`
Expected: PASS

- [ ] **Step 3: 커밋**

```bash
git add internal/comment/service_db_test.go
git commit -m "test: 댓글 생성/수정/삭제 접근 제어 테스트 (경로 8-10)"
```

---

### Task 8: User — Login/Register (경로 11·12)

**Files:**
- Create: `internal/user/service_db_test.go`

참고: `model.User`에는 email 필드가 없다 — 중복 검사는 username만 해당.
Register의 MySQL 1062 에러 분기는 SQLite에서 타지 않지만, 사전 중복 체크(`Where("username = ?").First`)가 먼저 걸리므로 테스트 가능.

- [ ] **Step 1: 테스트 작성**

`internal/user/service_db_test.go`:

```go
package user

import (
	"errors"
	"strings"
	"testing"

	"tolelom_api/internal/dto"
	"tolelom_api/internal/testutil"
)

// 테스트 전용 JWT 서명 키 — 실제 비밀값 아님 (런타임 생성)
var testJWTSecret = strings.Repeat("unit-test-key-", 4)

// 경로 11: 잘못된 비밀번호 차단 (bcrypt 해시 검증 경유)
func TestAuthenticateUser_PasswordVerification(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewAuthService(db, testJWTSecret)

	u := testutil.MakeUser(t, db, "alice")

	t.Run("잘못된 비밀번호 차단", func(t *testing.T) {
		_, err := svc.AuthenticateUser(&dto.LoginRequest{Username: "alice", Password: "wrong-password"})
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("err = %v, want ErrInvalidCredentials", err)
		}
	})

	t.Run("존재하지 않는 유저도 동일 에러 (정보 노출 방지)", func(t *testing.T) {
		_, err := svc.AuthenticateUser(&dto.LoginRequest{Username: "ghost", Password: "whatever"})
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("err = %v, want ErrInvalidCredentials", err)
		}
	})

	t.Run("올바른 비밀번호는 토큰 발급", func(t *testing.T) {
		res, err := svc.AuthenticateUser(&dto.LoginRequest{
			Username: "alice",
			Password: testutil.DefaultPassword,
		})
		if err != nil {
			t.Fatalf("로그인 실패: %v", err)
		}
		if res.AccessToken == "" || res.RefreshToken == "" {
			t.Fatal("토큰이 비어 있음")
		}
		if res.User.ID != u.ID {
			t.Fatalf("user.ID = %d, want %d", res.User.ID, u.ID)
		}
	})
}

// 경로 12: 중복 username 거부
func TestRegisterUser_DuplicateUsername(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewAuthService(db, testJWTSecret)

	first, err := svc.RegisterUser(&dto.RegisterRequest{Username: "newbie", Password: "register-pw-1!"})
	if err != nil {
		t.Fatalf("1차 가입 실패: %v", err)
	}
	if first.AccessToken == "" {
		t.Fatal("가입 시 토큰 미발급")
	}

	t.Run("동일 username 재가입 거부", func(t *testing.T) {
		_, err := svc.RegisterUser(&dto.RegisterRequest{Username: "newbie", Password: "register-pw-2!"})
		if !errors.Is(err, ErrUserAlreadyExists) {
			t.Fatalf("err = %v, want ErrUserAlreadyExists", err)
		}
	})

	t.Run("대소문자/공백 정규화 후에도 중복 거부", func(t *testing.T) {
		_, err := svc.RegisterUser(&dto.RegisterRequest{Username: "  NewBie  ", Password: "register-pw-2!"})
		if !errors.Is(err, ErrUserAlreadyExists) {
			t.Fatalf("err = %v, want ErrUserAlreadyExists (정규화 누락?)", err)
		}
	})
}
```

- [ ] **Step 2: 통과 확인**

Run: `go test ./internal/user/ -run 'TestAuthenticateUser_PasswordVerification|TestRegisterUser_DuplicateUsername' -v`
Expected: PASS

- [ ] **Step 3: 커밋**

```bash
git add internal/user/service_db_test.go
git commit -m "test: 로그인 비밀번호 검증·중복 가입 거부 테스트 (경로 11-12)"
```

---

### Task 9: 전체 검증 + 마무리

**Files:** 없음 (검증만)

- [ ] **Step 1: 전체 테스트 + race**

Run: `go test -race ./...`
Expected: 전 패키지 ok. (GetPostByID의 view count 고루틴 등에서 race 잡히면 보고)

- [ ] **Step 2: 커버리지 확인**

Run: `go test -cover ./internal/post/ ./internal/comment/ ./internal/user/ ./internal/testutil/`
Expected: post/comment/user 모두 시작점(40.7%/56.0%/47.5%) 대비 상승. 수치를 기록해 보고.

- [ ] **Step 3: 린트**

Run: `golangci-lint run`
Expected: 이슈 0건 (errcheck 주의 — 테스트 내 Count 쿼리 등 에러 무시는 명시 처리)

- [ ] **Step 4: WIP 스펙 문서 정리 커밋**

최종 스펙(2026-06-04)이 WIP를 대체하므로:

```bash
git rm docs/superpowers/specs/2026-04-25-backend-critical-path-tests-design-WIP.md
git add docs/superpowers/specs/2026-06-04-backend-critical-path-tests-design.md docs/superpowers/plans/2026-06-04-backend-critical-path-tests.md
git commit -m "docs: 크리티컬 패스 테스트 설계 최종본으로 WIP 대체"
```
