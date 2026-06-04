package testutil

import (
	"testing"

	"tolelom_api/internal/model"
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
	// DB에 실제로 false가 저장됐는지 재조회로 잠금 (GORM zero-value 함정 회귀 방지)
	var reloadedPriv model.Post
	if err := db.First(&reloadedPriv, priv.ID).Error; err != nil {
		t.Fatalf("reload 실패: %v", err)
	}
	if reloadedPriv.IsPublic {
		t.Fatal("DB에 is_public=true로 저장됨 — UpdateColumn 우회 깨짐")
	}

	draft := MakePost(t, db, u.ID, WithStatus("draft"))
	if draft.Status != "draft" {
		t.Fatalf("WithStatus 적용 안 됨: %q", draft.Status)
	}
	// DB에 실제로 draft가 저장됐는지 재조회로 잠금
	var reloadedDraft model.Post
	if err := db.First(&reloadedDraft, draft.ID).Error; err != nil {
		t.Fatalf("reload 실패: %v", err)
	}
	if reloadedDraft.Status != "draft" {
		t.Fatalf("DB에 status=%q로 저장됨 — WithStatus 미반영", reloadedDraft.Status)
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
