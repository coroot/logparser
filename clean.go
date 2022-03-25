package logparser

import (
	"regexp"
	"strings"
)

const (
	lookForTimestampLimit = 30
)

var (
	timestampRegexes = []*regexp.Regexp{
		regexp.MustCompile(`(^|\s)\d{2}:\d{2}(:\d{2}[^\s"']*)?`),
		regexp.MustCompile(`\d{2} [A-Z][a-z]{2} \d{4}`),
		regexp.MustCompile(`\d{4}-\d{2}-\d{2}`),
		regexp.MustCompile(`\d{4}/\d{2}/\d{2}`),
		regexp.MustCompile(`\d{4}\.\d{2}\.\d{2}`),
		regexp.MustCompile(`[A-Z][a-z]{2} \d{2}`),
		regexp.MustCompile(`\d{2}-\d{2}-\d{4}`),
		regexp.MustCompile(`\d{2}/\d{2}/\d{4}`),
		regexp.MustCompile(`\d{2}\.\d{2}\.\d{4}`),
	}
	extraSpaces = regexp.MustCompile(`\s+`)
)

func clean(line string) string {
	for _, r := range timestampRegexes {
		line = r.ReplaceAllString(line, "")
	}
	return strings.TrimSpace(extraSpaces.ReplaceAllString(line, " "))
}

func containsTimestamp(line string) bool {
	if len(line) > lookForTimestampLimit {
		line = line[:lookForTimestampLimit]
	}
	for _, re := range timestampRegexes {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}
