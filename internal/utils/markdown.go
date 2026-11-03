package utils

import (
	"regexp"
	"strings"
)

var markdownRe = regexp.MustCompile(`(?m)^#{1,6}\s+|[*_~` + "`" + `\[\]()!>|]|\{[^}]*\}`)

func StripMarkdown(text string) string {
	result := markdownRe.ReplaceAllString(text, "")
	result = strings.ReplaceAll(result, "\n", " ")
	return strings.Join(strings.Fields(result), " ")
}

// Excerpt is the markdown-stripped content cut to maxRunes, with "..." when cut.
func Excerpt(content string, maxRunes int) string {
	plain := []rune(StripMarkdown(content))
	if len(plain) <= maxRunes {
		return string(plain)
	}
	return string(plain[:maxRunes]) + "..."
}
