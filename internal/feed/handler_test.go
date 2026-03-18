package feed

import "testing"

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
		result := stripMarkdown(tt.input)
		if result != tt.expected {
			t.Errorf("stripMarkdown(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExcerpt(t *testing.T) {
	short := "짧은 글"
	result := excerpt(short, 200)
	if result != short {
		t.Errorf("excerpt short text = %q, want %q", result, short)
	}

	long := ""
	for i := 0; i < 250; i++ {
		long += "가"
	}
	result = excerpt(long, 200)
	runes := []rune(result)
	// 200 chars + "..." (3 chars)
	if len(runes) != 203 {
		t.Errorf("excerpt long text rune length = %d, want 203", len(runes))
	}
}

func TestUintToStr(t *testing.T) {
	tests := []struct {
		input    uint
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{12345, "12345"},
	}
	for _, tt := range tests {
		result := uintToStr(tt.input)
		if result != tt.expected {
			t.Errorf("uintToStr(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
