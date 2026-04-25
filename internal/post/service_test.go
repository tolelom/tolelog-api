package post

import (
	"strings"
	"testing"

	"gorm.io/gorm"
)

// --- NewService ---

func TestNewService(t *testing.T) {
	db := &gorm.DB{}
	svc := NewService(db, nil)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

// --- SanitizeSearchQuery ---

func TestSanitizeSearchQuery_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"golang", "golang"},
		{"  go lang  ", "go lang"},
		{"한글 검색어", "한글 검색어"},
		{"hello, world!", "hello, world!"},
		{"test (query)", "test (query)"},
	}
	for _, tt := range tests {
		result, err := SanitizeSearchQuery(tt.input)
		if err != nil {
			t.Errorf("SanitizeSearchQuery(%q) unexpected error: %v", tt.input, err)
		}
		if result != tt.expected {
			t.Errorf("SanitizeSearchQuery(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeSearchQuery_TooShort(t *testing.T) {
	_, err := SanitizeSearchQuery("a")
	if err == nil {
		t.Error("expected error for single-char query")
	}
}

func TestSanitizeSearchQuery_Empty(t *testing.T) {
	_, err := SanitizeSearchQuery("")
	if err == nil {
		t.Error("expected error for empty query")
	}
}

func TestSanitizeSearchQuery_TooLong(t *testing.T) {
	long := strings.Repeat("a", 101)
	_, err := SanitizeSearchQuery(long)
	if err == nil {
		t.Error("expected error for query over 100 chars")
	}
}

func TestSanitizeSearchQuery_ExactlyMinLength(t *testing.T) {
	_, err := SanitizeSearchQuery("go")
	if err != nil {
		t.Errorf("expected no error for 2-char query, got: %v", err)
	}
}

func TestSanitizeSearchQuery_ExactlyMaxLength(t *testing.T) {
	exact := strings.Repeat("a", 100)
	result, err := SanitizeSearchQuery(exact)
	if err != nil {
		t.Errorf("expected no error for 100-char query, got: %v", err)
	}
	if result != exact {
		t.Error("result mismatch for 100-char query")
	}
}

func TestSanitizeSearchQuery_InvalidChars(t *testing.T) {
	invalids := []string{
		"search%",
		"query`backtick",
		"pipe|test",
		"amp&test",
		"dollar$sign",
	}
	for _, q := range invalids {
		_, err := SanitizeSearchQuery(q)
		if err == nil {
			t.Errorf("SanitizeSearchQuery(%q) should reject invalid chars", q)
		}
	}
}

func TestSanitizeTag_ValidTags(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"golang", "golang"},
		{"Go", "Go"},
		{"c++", "c++"},
		{"c#", "c#"},
		{"node.js", "node.js"},
		{"한글태그", "한글태그"},
		{"  spacey  ", "spacey"},
		{"web-dev", "web-dev"},
		{"my_tag", "my_tag"},
	}

	for _, tt := range tests {
		result, err := SanitizeTag(tt.input)
		if err != nil {
			t.Errorf("SanitizeTag(%q) returned unexpected error: %v", tt.input, err)
		}
		if result != tt.expected {
			t.Errorf("SanitizeTag(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeTag_EmptyString(t *testing.T) {
	result, err := SanitizeTag("")
	if err != nil {
		t.Errorf("SanitizeTag('') should not error, got: %v", err)
	}
	if result != "" {
		t.Errorf("SanitizeTag('') = %q, want empty", result)
	}
}

func TestSanitizeTag_WhitespaceOnly(t *testing.T) {
	result, err := SanitizeTag("   ")
	if err != nil {
		t.Errorf("SanitizeTag('   ') should not error, got: %v", err)
	}
	if result != "" {
		t.Errorf("SanitizeTag('   ') = %q, want empty", result)
	}
}

func TestSanitizeTag_TooLong(t *testing.T) {
	longTag := ""
	for i := 0; i < 51; i++ {
		longTag += "a"
	}

	_, err := SanitizeTag(longTag)
	if err == nil {
		t.Error("SanitizeTag should reject tags longer than 50 characters")
	}
}

func TestSanitizeTag_InvalidCharacters(t *testing.T) {
	invalidTags := []string{
		"tag;DROP TABLE",
		"tag' OR 1=1",
		"tag<script>",
		"tag$(cmd)",
		"tag`echo`",
		"tag|pipe",
		"tag&amp",
	}

	for _, tag := range invalidTags {
		_, err := SanitizeTag(tag)
		if err == nil {
			t.Errorf("SanitizeTag(%q) should reject invalid characters", tag)
		}
	}
}

func TestSanitizeTag_ExactlyMaxLength(t *testing.T) {
	tag50 := ""
	for i := 0; i < 50; i++ {
		tag50 += "a"
	}

	result, err := SanitizeTag(tag50)
	if err != nil {
		t.Errorf("SanitizeTag with 50 chars should succeed, got: %v", err)
	}
	if result != tag50 {
		t.Errorf("SanitizeTag result mismatch")
	}
}
