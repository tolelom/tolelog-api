package utils

import (
	"strings"
	"testing"
)

func TestStripMarkdown(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"# Hello World", "Hello World"},
		{"**bold** text", "bold text"},
		{"normal text", "normal text"},
		{"", ""},
	}
	for _, tt := range tests {
		result := StripMarkdown(tt.input)
		if result != tt.expected {
			t.Errorf("StripMarkdown(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExcerpt(t *testing.T) {
	short := "짧은 글"
	result := Excerpt(short, 200)
	if result != short {
		t.Errorf("excerpt short text = %q, want %q", result, short)
	}

	long := strings.Repeat("가", 250)
	result = Excerpt(long, 200)
	runes := []rune(result)
	if len(runes) != 203 {
		t.Errorf("excerpt long text rune length = %d, want 203", len(runes))
	}
}
