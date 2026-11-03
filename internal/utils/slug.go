package utils

import (
	"strconv"
	"strings"
	"unicode"
)

// slugMaxRunes 는 URL 을 짧게 두면서 한글 제목 대부분을 담는 길이다.
const slugMaxRunes = 60

// Slugify 는 글자·숫자만 소문자로 남기고 공백·_·- 를 하이픈 하나로 바꾼다. 남는 글자가 없으면 "" 다.
func Slugify(title string) string {
	var b strings.Builder
	b.Grow(len(title))

	lastWasHyphen := false
	for _, r := range strings.TrimSpace(title) {
		switch {
		case unicode.IsDigit(r):
			b.WriteRune(r)
			lastWasHyphen = false
		case unicode.IsLetter(r):
			b.WriteRune(unicode.ToLower(r))
			lastWasHyphen = false
		case r == ' ' || r == '\t' || r == '\n' || r == '_' || r == '-':
			if b.Len() > 0 && !lastWasHyphen {
				b.WriteRune('-')
				lastWasHyphen = true
			}
		}
	}

	s := strings.TrimRight(b.String(), "-")
	runes := []rune(s)
	if len(runes) > slugMaxRunes {
		s = strings.TrimRight(string(runes[:slugMaxRunes]), "-")
	}
	return s
}

// PostSlugPath is the canonical post path /post/{slug}-{id}, or /post/{id} when the title yields no slug.
func PostSlugPath(title string, id uint) string {
	idStr := strconv.FormatUint(uint64(id), 10)
	if slug := Slugify(title); slug != "" {
		return "/post/" + slug + "-" + idStr
	}
	return "/post/" + idStr
}
