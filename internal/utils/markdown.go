package utils

import (
	"regexp"
	"strings"
)

var markdownRe = regexp.MustCompile(`(?m)^#{1,6}\s+|[*_~` + "`" + `\[\]()!>|]|\{[^}]*\}`)

// StripMarkdown removes common markdown syntax characters from text.
func StripMarkdown(text string) string {
	result := markdownRe.ReplaceAllString(text, "")
	result = strings.ReplaceAll(result, "\n", " ")
	return strings.Join(strings.Fields(result), " ")
}
