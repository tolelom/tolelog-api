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

func login(t *testing.T, svc AuthService, username, password string) (*dto.AuthResponse, error) {
	t.Helper()
	return svc.AuthenticateUser(&dto.LoginRequest{Username: username, Password: password})
}

func TestRegisterUser_NormalizesUsernameAndIssuesTokens(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewAuthService(db, testJWTSecret)

	res, err := svc.RegisterUser(&dto.RegisterRequest{Username: "  Alice ", Password: testutil.DefaultPassword})
	if err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}
	if res.User.Username != "alice" || res.AccessToken == "" || res.RefreshToken == "" {
		t.Fatalf("res = %+v, want username alice and both tokens", res)
	}
	if _, err := login(t, svc, "ALICE", testutil.DefaultPassword); err != nil {
		t.Fatalf("대소문자만 다른 아이디로 로그인 실패: %v", err)
	}
	if _, err := svc.RegisterUser(&dto.RegisterRequest{Username: "ALICE", Password: testutil.DefaultPassword}); !errors.Is(err, ErrUserAlreadyExists) {
		t.Fatalf("대소문자만 다른 아이디 가입 err = %v, want ErrUserAlreadyExists", err)
	}
}

func TestRefreshTokens(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewAuthService(db, testJWTSecret)
	u := testutil.MakeUser(t, db, "alice")
	session, err := login(t, svc, "alice", testutil.DefaultPassword)
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	t.Run("리프레시 토큰으로 새 토큰 쌍을 받는다", func(t *testing.T) {
		res, err := svc.RefreshTokens(session.RefreshToken)
		if err != nil {
			t.Fatalf("RefreshTokens: %v", err)
		}
		if res.User.ID != u.ID || res.AccessToken == "" || res.RefreshToken == "" {
			t.Fatalf("res = %+v", res)
		}
	})

	t.Run("액세스 토큰은 리프레시에 쓸 수 없다", func(t *testing.T) {
		if _, err := svc.RefreshTokens(session.AccessToken); err == nil {
			t.Fatal("액세스 토큰으로 리프레시가 됐다")
		}
	})

	t.Run("다른 키로 서명한 토큰은 거부한다", func(t *testing.T) {
		other := NewAuthService(db, strings.Repeat("other-key-", 6))
		if _, err := other.RefreshTokens(session.RefreshToken); err == nil {
			t.Fatal("다른 서명 키의 토큰이 통과했다")
		}
	})

	t.Run("로그아웃하면 이전 리프레시 토큰을 거부한다", func(t *testing.T) {
		if err := svc.Logout(u.ID); err != nil {
			t.Fatalf("Logout: %v", err)
		}
		if _, err := svc.RefreshTokens(session.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf("err = %v, want ErrInvalidRefreshToken", err)
		}
		fresh, err := login(t, svc, "alice", testutil.DefaultPassword)
		if err != nil {
			t.Fatalf("다시 로그인: %v", err)
		}
		if _, err := svc.RefreshTokens(fresh.RefreshToken); err != nil {
			t.Fatalf("다시 로그인한 토큰 리프레시 실패: %v", err)
		}
	})

	t.Run("탈퇴한 사용자의 토큰은 거부한다", func(t *testing.T) {
		fresh, _ := login(t, svc, "alice", testutil.DefaultPassword)
		if err := svc.DeleteUser(u.ID); err != nil {
			t.Fatalf("DeleteUser: %v", err)
		}
		if _, err := svc.RefreshTokens(fresh.RefreshToken); !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("err = %v, want ErrUserNotFound", err)
		}
		if _, err := login(t, svc, "alice", testutil.DefaultPassword); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("탈퇴 뒤 로그인 err = %v, want ErrInvalidCredentials", err)
		}
	})
}

func TestChangePassword(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewAuthService(db, testJWTSecret)
	u := testutil.MakeUser(t, db, "alice")
	const newPassword = "fixture-new-pw-2!"

	t.Run("현재 비밀번호가 틀리면 거부한다", func(t *testing.T) {
		_, err := svc.ChangePassword(u.ID, &dto.ChangePasswordRequest{CurrentPassword: "wrong-pw", NewPassword: newPassword})
		if !errors.Is(err, ErrInvalidPassword) {
			t.Fatalf("err = %v, want ErrInvalidPassword", err)
		}
	})

	t.Run("같은 비밀번호로는 바꾸지 않는다", func(t *testing.T) {
		_, err := svc.ChangePassword(u.ID, &dto.ChangePasswordRequest{CurrentPassword: testutil.DefaultPassword, NewPassword: testutil.DefaultPassword})
		if !errors.Is(err, ErrSamePassword) {
			t.Fatalf("err = %v, want ErrSamePassword", err)
		}
	})

	t.Run("바꾸면 새 비밀번호로만 로그인되고 다른 세션은 끊긴다", func(t *testing.T) {
		otherDevice, err := login(t, svc, "alice", testutil.DefaultPassword)
		if err != nil {
			t.Fatalf("login: %v", err)
		}
		res, err := svc.ChangePassword(u.ID, &dto.ChangePasswordRequest{CurrentPassword: testutil.DefaultPassword, NewPassword: newPassword})
		if err != nil {
			t.Fatalf("ChangePassword: %v", err)
		}
		if _, err := svc.RefreshTokens(otherDevice.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf("바꾸기 전 리프레시 토큰 err = %v, want ErrInvalidRefreshToken", err)
		}
		if _, err := svc.RefreshTokens(res.RefreshToken); err != nil {
			t.Fatalf("바꾼 기기가 받은 리프레시 토큰이 거부됐다: %v", err)
		}
		if res.User.ID != u.ID || res.AccessToken == "" {
			t.Fatalf("res = %+v", res)
		}
		if _, err := login(t, svc, "alice", testutil.DefaultPassword); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("옛 비밀번호 로그인 err = %v, want ErrInvalidCredentials", err)
		}
		if _, err := login(t, svc, "alice", newPassword); err != nil {
			t.Fatalf("새 비밀번호 로그인 실패: %v", err)
		}
	})

	t.Run("없는 사용자", func(t *testing.T) {
		_, err := svc.ChangePassword(9999, &dto.ChangePasswordRequest{CurrentPassword: "x", NewPassword: newPassword})
		if !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("err = %v, want ErrUserNotFound", err)
		}
	})
}

func TestGetUserByIDAndUpdateAvatar(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewAuthService(db, testJWTSecret)
	u := testutil.MakeUser(t, db, "alice")

	if err := svc.UpdateAvatar(u.ID, "/uploads/images/a.png"); err != nil {
		t.Fatalf("UpdateAvatar: %v", err)
	}
	got, err := svc.GetUserByID(u.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got.AvatarURL != "/uploads/images/a.png" {
		t.Fatalf("AvatarURL = %q", got.AvatarURL)
	}
	if _, err := svc.GetUserByID(9999); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("err = %v, want ErrUserNotFound", err)
	}
}
