package sitemap

import (
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

// mockRepo implements Repository for testing.
type mockRepo struct {
	postsFn  func() ([]Entry, error)
	seriesFn func() ([]Entry, error)
}

func (m *mockRepo) GetPublicPostEntries() ([]Entry, error) {
	if m.postsFn != nil {
		return m.postsFn()
	}
	return nil, nil
}

func (m *mockRepo) GetSeriesEntries() ([]Entry, error) {
	if m.seriesFn != nil {
		return m.seriesFn()
	}
	return nil, nil
}

func setupSitemapApp(repo Repository) *fiber.App {
	app := fiber.New()
	h := NewHandler(repo)
	app.Get("/sitemap.xml", h.Sitemap)
	return app
}

func TestSitemap_Success(t *testing.T) {
	now := time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		postsFn:  func() ([]Entry, error) { return []Entry{{ID: 1, Title: "Hello World", UpdatedAt: now}}, nil },
		seriesFn: func() ([]Entry, error) { return []Entry{{ID: 2, UpdatedAt: now}}, nil },
	}
	app := setupSitemapApp(repo)

	req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	xmlStr := string(body)
	// 슬러그 URL 형태: /post/{slug}-{id}
	if !containsStr(xmlStr, "https://blog.tolelom.xyz/post/hello-world-1") {
		t.Errorf("missing slug post URL in sitemap; body: %s", xmlStr)
	}
	if !containsStr(xmlStr, "https://blog.tolelom.xyz/series/2") {
		t.Error("missing series URL in sitemap")
	}
}

func TestSitemap_PostsError(t *testing.T) {
	repo := &mockRepo{
		postsFn: func() ([]Entry, error) { return nil, errors.New("db error") },
	}
	app := setupSitemapApp(repo)

	req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}
}

func TestSitemap_SeriesError(t *testing.T) {
	repo := &mockRepo{
		postsFn:  func() ([]Entry, error) { return nil, nil },
		seriesFn: func() ([]Entry, error) { return nil, errors.New("db error") },
	}
	app := setupSitemapApp(repo)

	req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}
}

func TestSitemap_ContentType(t *testing.T) {
	repo := &mockRepo{}
	app := setupSitemapApp(repo)

	req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	resp, _ := app.Test(req)
	ct := resp.Header.Get("Content-Type")
	if !containsStr(ct, "xml") {
		t.Errorf("Content-Type = %q, want xml", ct)
	}
}

func TestURLSetMarshal(t *testing.T) {
	set := urlSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs: []siteURL_{
			{Loc: "https://blog.tolelom.xyz", LastMod: "2026-03-18"},
			{Loc: "https://blog.tolelom.xyz/post/hello-1", LastMod: "2026-03-17"},
		},
	}

	output, err := xml.MarshalIndent(set, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent failed: %v", err)
	}

	xmlStr := string(output)

	if !containsStr(xmlStr, "http://www.sitemaps.org/schemas/sitemap/0.9") {
		t.Error("missing sitemap namespace in output")
	}
	if !containsStr(xmlStr, "<loc>https://blog.tolelom.xyz</loc>") {
		t.Error("missing root URL in output")
	}
	if !containsStr(xmlStr, "<loc>https://blog.tolelom.xyz/post/hello-1</loc>") {
		t.Error("missing post URL in output")
	}
	if !containsStr(xmlStr, "<lastmod>2026-03-18</lastmod>") {
		t.Error("missing lastmod in output")
	}
}

func TestURLSetEmpty(t *testing.T) {
	set := urlSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  []siteURL_{},
	}

	_, err := xml.MarshalIndent(set, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent failed for empty urlset: %v", err)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
