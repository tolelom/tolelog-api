package router

import (
	"net/http"
	"os"
	"regexp"
	"testing"
)

// Dockerfile HEALTHCHECK 가 부르는 경로가 실제 라우트여야 컨테이너가 healthy 가 된다.
func TestDockerfileHealthcheckHitsARoute(t *testing.T) {
	raw, err := os.ReadFile("../../Dockerfile")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`curl -f http://localhost:\d+(/\S*)`).FindSubmatch(raw)
	if m == nil {
		t.Fatal("HEALTHCHECK curl line not found in Dockerfile")
	}
	path := string(m[1])
	app, _ := newFlowApp(t)
	if got := doRequest(t, app, http.MethodGet, path, "", ""); got != http.StatusOK {
		t.Errorf("GET %s = %d, want 200", path, got)
	}
}
