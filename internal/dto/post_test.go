package dto

import (
	"testing"
	"tolelom_api/internal/utils"
)

func TestMakeExcerpt(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantFull bool // true if result should equal stripped content (no truncation)
	}{
		{"empty", "", true},
		{"short plain text", "짧은 글입니다", true},
		{"strips markdown headings", "# Hello World", true},
		{"strips bold", "**bold** text", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := makeExcerpt(tt.content)
			if tt.wantFull {
				stripped := utils.StripMarkdown(tt.content)
				if result != stripped {
					t.Errorf("makeExcerpt(%q) = %q, want %q", tt.content, result, stripped)
				}
			}
		})
	}
}

func TestMakeExcerptTruncation(t *testing.T) {
	long := ""
	for i := 0; i < 250; i++ {
		long += "가"
	}
	result := makeExcerpt(long)
	runes := []rune(result)
	// 200 chars + "..." (3 chars)
	if len(runes) != 203 {
		t.Errorf("makeExcerpt long text rune length = %d, want 203", len(runes))
	}
}
