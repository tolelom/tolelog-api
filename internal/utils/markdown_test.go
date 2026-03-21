package utils

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
		result := StripMarkdown(tt.input)
		if result != tt.expected {
			t.Errorf("StripMarkdown(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
