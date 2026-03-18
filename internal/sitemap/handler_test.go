package sitemap

import (
	"encoding/xml"
	"testing"
)

func TestURLSetMarshal(t *testing.T) {
	set := urlSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs: []siteURL_{
			{Loc: "https://tolelom.xyz", LastMod: "2026-03-18"},
			{Loc: "https://tolelom.xyz/post/1", LastMod: "2026-03-17"},
		},
	}

	output, err := xml.MarshalIndent(set, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent failed: %v", err)
	}

	xmlStr := string(output)

	// Verify root element and namespace are present
	if !containsStr(xmlStr, "http://www.sitemaps.org/schemas/sitemap/0.9") {
		t.Error("missing sitemap namespace in output")
	}

	// Verify URLs are present
	if !containsStr(xmlStr, "<loc>https://tolelom.xyz</loc>") {
		t.Error("missing root URL in output")
	}
	if !containsStr(xmlStr, "<loc>https://tolelom.xyz/post/1</loc>") {
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
