package logparser

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"regexp"
	"strings"
)

const (
	minWordLen = 2
)

var (
	squote  = '\''
	dquote  = '"'
	bslash  = '\\'
	lsbrack = '['
	rsbrack = ']'
	lpar    = '('
	rpar    = ')'
	lcur    = '{'
	rcur    = '}'

	hex  = regexp.MustCompile(`^[a-fA-F0-9]{4,}$`)
	uuid = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)
	num  = regexp.MustCompile(`\d+`)
	word = regexp.MustCompile(`^[a-zA-Z][a-zA-Z\._-]*[a-zA-Z]$`)
)

type Pattern struct {
	words []string
	str   *string
	hash  *string
}

func (p *Pattern) String() string {
	if p.str == nil {
		s := strings.Join(p.words, " ")
		p.str = &s
	}
	return *p.str
}

func (p *Pattern) Hash() string {
	if p.hash == nil {
		h := fmt.Sprintf("%x", md5.Sum([]byte(p.String())))
		p.hash = &h
	}
	return *p.hash
}

func (p *Pattern) WeakEqual(other *Pattern) bool {
	if len(p.words) != len(other.words) {
		return false
	}
	var matches int
	for i, op := range other.words {
		if p.words[i] == op {
			matches++
		}
	}
	if matches >= len(p.words)-1 {
		return true
	}
	return false
}

func NewPattern(input string) *Pattern {
	pattern := &Pattern{}
	for _, p := range strings.Fields(removeQuotedAndBrackets(input)) {
		p = strings.TrimRight(p, "=:],;")
		if len(p) < minWordLen {
			continue
		}
		if hex.MatchString(p) || uuid.MatchString(p) {
			continue
		}
		p = num.ReplaceAllLiteralString(p, "")
		if word.MatchString(p) {
			pattern.words = append(pattern.words, p)
		}
	}
	return pattern
}

func removeQuotedAndBrackets(data string) string {
	var res bytes.Buffer
	var quote, prev rune
	var seenBrackets []rune
	var l int
	for i, r := range data {
		switch r {
		case lsbrack, lpar, lcur:
			if quote == 0 {
				seenBrackets = append(seenBrackets, r)
			}
		case rsbrack:
			if l = len(seenBrackets); l > 0 && seenBrackets[l-1] == lsbrack {
				seenBrackets = seenBrackets[:l-1]
				continue
			}
		case rpar:
			if l = len(seenBrackets); l > 0 && seenBrackets[l-1] == lpar {
				seenBrackets = seenBrackets[:l-1]
				continue
			}
		case rcur:
			if l = len(seenBrackets); l > 0 && seenBrackets[l-1] == lcur {
				seenBrackets = seenBrackets[:l-1]
				continue
			}
		case dquote, squote:
			prev = 0
			if i > 0 {
				prev = rune(data[i-1])
			}
			if prev != bslash && len(seenBrackets) == 0 {
				if quote == 0 {
					quote = r
				} else if quote == r {
					quote = 0
					continue
				}
			}
		}
		if quote != 0 || len(seenBrackets) > 0 {
			continue
		}
		res.WriteRune(r)
	}
	return res.String()
}
