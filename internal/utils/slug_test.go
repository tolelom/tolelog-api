package utils

import "testing"

func TestSlugify(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"whitespace", "   ", ""},
		{"english", "Hello World", "hello-world"},
		{"lowercase already", "hello world", "hello-world"},
		{"mixed case", "Hello WORLD foo", "hello-world-foo"},
		{"digits preserved", "Version 2.0 release", "version-20-release"},
		{"korean", "리액트 마크다운 블로그", "리액트-마크다운-블로그"},
		{"korean mixed english", "React로 만든 블로그", "react로-만든-블로그"},
		{"emoji removed", "안녕 👋 세상", "안녕-세상"},
		{"punctuation removed", "What?! Why??", "what-why"},
		{"underscores to hyphen", "foo_bar_baz", "foo-bar-baz"},
		{"multi space collapsed", "foo   bar    baz", "foo-bar-baz"},
		{"leading trailing hyphen trimmed", "  -hello-  ", "hello"},
		{"all special chars", "!@#$%^&*()", ""},
		{"newline as space", "first line\nsecond", "first-line-second"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Slugify(tc.in)
			if got != tc.want {
				t.Errorf("Slugify(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSlugifyTruncation(t *testing.T) {
	// 60 룬 초과는 자르고, 잘린 끝의 하이픈은 정리.
	long := ""
	for i := 0; i < 80; i++ {
		long += "가"
	}
	got := Slugify(long)
	if r := []rune(got); len(r) > 60 {
		t.Errorf("expected len ≤ 60, got %d", len(r))
	}
}

func TestPostSlugPath(t *testing.T) {
	cases := []struct {
		title string
		id    uint
		want  string
	}{
		{"Hello World", 123, "/post/hello-world-123"},
		{"리액트 블로그", 1, "/post/리액트-블로그-1"},
		{"!@#$%", 42, "/post/42"}, // 슬러그 없으면 id 만
		{"", 7, "/post/7"},
	}
	for _, tc := range cases {
		got := PostSlugPath(tc.title, tc.id)
		if got != tc.want {
			t.Errorf("PostSlugPath(%q,%d) = %q, want %q", tc.title, tc.id, got, tc.want)
		}
	}
}
