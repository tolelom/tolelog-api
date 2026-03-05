package post

import "testing"

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
