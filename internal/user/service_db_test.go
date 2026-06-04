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
// 서브테스트는 순차 시나리오 (1차 가입 → 중복 거부) — 단독 실행 불가.
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
