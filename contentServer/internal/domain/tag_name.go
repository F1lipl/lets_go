package domain

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

const MaxPostTags = 10

type TagName struct {
	Normalized string
	Display    string
}

// PrepareTagNames keeps the author's order while collapsing equivalent names.
func PrepareTagNames(names []string) ([]TagName, error) {
	if len(names) > MaxPostTags {
		return nil, ErrTooManyTags
	}
	result := make([]TagName, 0, len(names))
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		display := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(name), "#"))
		display = norm.NFKC.String(display)
		normalized := cases.Fold().String(display)
		if display == "" || utf8.RuneCountInString(display) > 64 ||
			utf8.RuneCountInString(normalized) > 64 || strings.ContainsRune(display, '#') ||
			strings.IndexFunc(display, unicode.IsControl) >= 0 {
			return nil, ErrTagNameInvalid
		}
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		result = append(result, TagName{Normalized: normalized, Display: display})
	}
	return result, nil
}
