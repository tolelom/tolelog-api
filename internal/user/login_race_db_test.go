package user

import (
	"testing"

	"gorm.io/gorm"

	"tolelom_api/internal/dto"
	"tolelom_api/internal/model"
	"tolelom_api/internal/testutil"
)

// 로그인이 사용자 행을 읽은 뒤 쓰기 직전에, 다른 기기의 로그아웃이 커밋된 상황을 만든다.
func TestAuthenticateUser_DoesNotRevertConcurrentLogout(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewAuthService(db, testJWTSecret)
	u := testutil.MakeUser(t, db, "alice")

	fired := false
	err := db.Callback().Update().Before("gorm:update").Register("test:concurrent_logout", func(tx *gorm.DB) {
		if fired || tx.Statement.Table != "users" {
			return
		}
		fired = true
		tx.Session(&gorm.Session{NewDB: true}).
			Exec("UPDATE users SET token_version = token_version + 1 WHERE id = ?", u.ID)
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.AuthenticateUser(&dto.LoginRequest{Username: "alice", Password: testutil.DefaultPassword}); err != nil {
		t.Fatalf("로그인 실패: %v", err)
	}
	if !fired {
		t.Fatal("겹친 로그아웃이 주입되지 않았다")
	}

	var got model.User
	if err := db.First(&got, u.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.TokenVersion != 1 {
		t.Fatalf("token_version = %d, want 1 (로그인이 겹친 로그아웃을 되돌렸다)", got.TokenVersion)
	}
}
