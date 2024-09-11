package logparser

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"regexp"
	"strings"
)

const (
	patternMaxWords  = 100
	patterMinWordLen = 2
	patternMaxDiff   = 1
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
	var diffs int
	for i := range other.words {
		if p.words[i] != other.words[i] {
			diffs++
			if diffs > patternMaxDiff {
				return false
			}
		}
	}
	return true
}

func NewPattern(input string) *Pattern {
	pattern := &Pattern{}
	for _, p := range strings.Fields(removeQuotedAndBrackets(input)) {
		p = strings.TrimRight(p, "=:],;")
		if len(p) < patterMinWordLen {
			continue
		}
		if hex.MatchString(p) || uuid.MatchString(p) {
			continue
		}
		p = removeDigits(p)
		if !isWord(p) {
			continue
		}
		pattern.words = append(pattern.words, p)
		if len(pattern.words) >= patternMaxWords {
			break
		}
	}
	return pattern
}

func NewPatternFromWords(input string) *Pattern {
	return &Pattern{words: strings.Split(input, " ")}
}

// like regexp match to `^[a-zA-Z][a-zA-Z._-]*[a-zA-Z]$`, but much faster
func isWord(s string) bool {
	l := len(s) - 1
	var firstLast int
	for i, r := range s {
		switch i {
		case 0, l:
			switch {
			case r >= 'A' && r <= 'Z':
				firstLast++
			case r >= 'a' && r <= 'z':
				firstLast++
			default:
				return false
			}
		default:
			switch {
			case r >= 'A' && r <= 'Z':
			case r >= 'a' && r <= 'z':
			case r == '.':
			case r == '_':
			case r == '-':
			default:
				return false
			}
		}
	}
	return firstLast == 2
}

func removeDigits(s string) string {
	res := bytes.NewBufferString(s)
	res.Reset()
	for _, r := range s {
		if r >= '0' && r <= '9' {
			continue
		}
		res.WriteRune(r)
	}
	return res.String()
}

func removeQuotedAndBrackets(s string) string {
	res := bytes.NewBufferString(s)
	res.Reset()
	var quote, prev rune
	var seenBrackets []rune
	var l int
	for i, r := range s {
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
				prev = rune(s[i-1])
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
