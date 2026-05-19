package utils

import (
	"strings"
	"unicode"
)

// slugMaxRunes 는 슬러그가 가질 수 있는 최대 룬(코드 포인트) 수.
// URL 길이를 합리적으로 유지하면서 한글 제목 대부분을 보존할 수 있는 값.
const slugMaxRunes = 60

// Slugify 는 제목을 URL-safe 슬러그로 변환한다.
// 보존: 영문 소문자/숫자/한글(가-힣)/하이픈.
// 변환: 영문 대문자→소문자, 공백/언더스코어→하이픈, 연속 하이픈→단일 하이픈.
// 제거: 그 외 모든 문자(특수문자/이모지/구두점).
// 빈 결과(예: 제목이 전부 특수문자) 인 경우 호출 측에서 id 만 사용하도록 빈 문자열을 반환한다.
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
		default:
			// 그 외 문자는 무시 (이모지, 구두점, URL-unsafe 등)
		}
	}

	s := strings.TrimRight(b.String(), "-")
	runes := []rune(s)
	if len(runes) > slugMaxRunes {
		s = strings.TrimRight(string(runes[:slugMaxRunes]), "-")
	}
	return s
}

// PostSlugPath 는 게시글의 canonical URL 경로(`/post/{slug}-{id}`)를 반환한다.
// 슬러그가 비어있으면 `/post/{id}` 형태로 폴백.
func PostSlugPath(title string, id uint) string {
	slug := Slugify(title)
	if slug == "" {
		return formatPostPath(id)
	}
	return "/post/" + slug + "-" + uintToString(id)
}

func formatPostPath(id uint) string {
	return "/post/" + uintToString(id)
}

func uintToString(n uint) string {
	// strconv.Itoa 사용 회피 — 작은 의존성 그래프 유지
	if n == 0 {
		return "0"
	}
	var digits [20]byte
	idx := len(digits)
	for n > 0 {
		idx--
		digits[idx] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[idx:])
}
