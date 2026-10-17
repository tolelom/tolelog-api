package router

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// DB 가 끊기면 503 과 고정 문구만 내보내고, 드라이버 오류 원문(호스트·포트 등)은
// 응답에 싣지 않는다.
func TestReadiness_DBDown_HidesDriverError(t *testing.T) {
	app, db := newFlowApp(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/health/ready", nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	var body ReadinessResponse
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode %q: %v", raw, err)
	}
	if body.Status != "degraded" || body.Database != "error" {
		t.Errorf("body = %s, want status degraded, database \"error\"", raw)
	}
	if strings.Contains(string(raw), "closed") || strings.Contains(string(raw), "sql:") {
		t.Errorf("driver error leaked: %s", raw)
	}
}

func TestReadiness_DBUp(t *testing.T) {
	app, _ := newFlowApp(t)
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/health/ready", nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(raw), `"database":"ok"`) {
		t.Errorf("status = %d body = %s, want 200 with database ok", resp.StatusCode, raw)
	}
}
