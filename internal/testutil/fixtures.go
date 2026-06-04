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

// postConfig collects desired post field values, including explicit bool overrides.
type postConfig struct {
	isPublic    bool
	setIsPublic bool // true when WithPrivate() was called
	status      string
	title       string
}

// PostOpt customizes a fixture post.
type PostOpt func(*postConfig)

// WithPrivate marks the post as private (is_public=false).
func WithPrivate() PostOpt {
	return func(c *postConfig) {
		c.isPublic = false
		c.setIsPublic = true
	}
}

// WithStatus sets the post status (e.g. "draft").
func WithStatus(status string) PostOpt {
	return func(c *postConfig) { c.status = status }
}

// WithTitle sets the post title.
func WithTitle(title string) PostOpt {
	return func(c *postConfig) { c.title = title }
}

// MakePost creates a published public post by default.
// GORM ignores zero-value booleans during INSERT (replacing them with the column
// default), so we create with the default value (true) and then apply any
// explicit bool overrides via UpdateColumn after creation.
func MakePost(t *testing.T, db *gorm.DB, userID uint, opts ...PostOpt) *model.Post {
	t.Helper()

	cfg := &postConfig{
		isPublic: true,
		status:   "published",
		title:    "테스트 글",
	}
	for _, opt := range opts {
		opt(cfg)
	}

	p := &model.Post{
		Title:    cfg.title,
		Content:  "본문",
		UserID:   userID,
		IsPublic: true, // always start true; override below if needed
		Status:   cfg.status,
	}
	if err := db.Create(p).Error; err != nil {
		t.Fatalf("post fixture 생성 실패: %v", err)
	}

	// Apply explicit bool overrides via raw column update to bypass GORM's
	// zero-value-omission behaviour for boolean columns with DB defaults.
	if cfg.setIsPublic && !cfg.isPublic {
		if err := db.Model(p).UpdateColumn("is_public", false).Error; err != nil {
			t.Fatalf("post fixture is_public 업데이트 실패: %v", err)
		}
		p.IsPublic = false
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
