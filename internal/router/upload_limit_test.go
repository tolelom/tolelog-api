package router

import (
	"net/http"
	"testing"
)

// 아바타 업로드도 업로드 한도(분당 20회)를 받는다. 한도는 /upload 와 함께 쓴다.
func TestAvatarUpload_IsRateLimited(t *testing.T) {
	app, _ := newFlowApp(t)
	for i := 1; i <= 20; i++ {
		if got := doRequest(t, app, http.MethodPut, "/api/v1/users/avatar", "", ""); got == http.StatusTooManyRequests {
			t.Fatalf("request %d already limited", i)
		}
	}
	if got := doRequest(t, app, http.MethodPut, "/api/v1/users/avatar", "", ""); got != http.StatusTooManyRequests {
		t.Errorf("21st avatar request = %d, want 429", got)
	}
}

func TestAvatarUpload_SharesUploadBudget(t *testing.T) {
	app, _ := newFlowApp(t)
	for i := 1; i <= 20; i++ {
		doRequest(t, app, http.MethodPost, "/api/v1/upload", "", "")
	}
	if got := doRequest(t, app, http.MethodPut, "/api/v1/users/avatar", "", ""); got != http.StatusTooManyRequests {
		t.Errorf("avatar after 20 uploads = %d, want 429", got)
	}
}
