package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"tolelom_api/internal/config"
	"tolelom_api/internal/testutil"
)

// app.Test 의 요청은 원격 주소가 0.0.0.0 으로 들어온다. 그 주소를 Traefik 처럼 신뢰 프록시로 둘지에 따라 나눠 본다.
func newProxiedApp(t *testing.T, trustedProxies []string) *fiber.App {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:      testJWTSecret,
		Environment:    "test",
		UploadDir:      t.TempDir(),
		DB:             testutil.SetupDB(t),
		TrustedProxies: trustedProxies,
	}
	app := fiber.New(AppConfig(cfg))
	Setup(app, cfg)
	return app
}

func loginFrom(t *testing.T, app *fiber.App, clientIP string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"nobody","password":"wrong-pass"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("CF-Connecting-IP", clientIP)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	return resp.StatusCode
}

func TestAuthRateLimit_IsPerClientBehindTrustedProxy(t *testing.T) {
	app := newProxiedApp(t, []string{"0.0.0.0"})

	for i := 0; i < 10; i++ {
		loginFrom(t, app, "203.0.113.10")
	}
	if got := loginFrom(t, app, "203.0.113.10"); got != http.StatusTooManyRequests {
		t.Fatalf("한도를 넘긴 클라이언트 status = %d, want 429", got)
	}
	if got := loginFrom(t, app, "198.51.100.20"); got == http.StatusTooManyRequests {
		t.Fatal("다른 클라이언트까지 막혔다 (프록시 IP 하나로 묶였다)")
	}
}

func TestAuthRateLimit_IgnoresClientIPHeaderFromUntrustedSource(t *testing.T) {
	app := newProxiedApp(t, []string{"10.0.0.1"})

	for i := 0; i < 10; i++ {
		loginFrom(t, app, "203.0.113.10")
	}
	if got := loginFrom(t, app, "198.51.100.20"); got != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429 (신뢰하지 않는 곳에서 온 헤더로 한도를 피하면 안 된다)", got)
	}
}
