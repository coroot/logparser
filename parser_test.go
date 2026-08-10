package logparser

import (
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestParserJson(t *testing.T) {
	line := `{"level":"error","msg":"payment failed","order_id":"ord-1"}`

	var gotLevel Level
	var gotMsg string
	var gotAttrs map[string]string
	cb := func(ts time.Time, level Level, patternHash string, msg string, attributes map[string]string) {
		gotLevel, gotMsg, gotAttrs = level, msg, attributes
	}

	p := &Parser{
		patterns:              map[patternKey]*patternStat{},
		patternsPerLevel:      map[Level]int{},
		patternsPerLevelLimit: 10,
		parseJson:             true,
		onMsgCb:               cb,
	}
	p.inc(Message{Timestamp: time.Now(), Content: line, Level: LevelUnknown})
	assert.Equal(t, LevelError, gotLevel)
	assert.Equal(t, "payment failed", gotMsg)
	assert.Equal(t, map[string]string{"order_id": "ord-1"}, gotAttrs)

	p = &Parser{
		patterns:              map[patternKey]*patternStat{},
		patternsPerLevel:      map[Level]int{},
		patternsPerLevelLimit: 10,
		onMsgCb:               cb,
	}
	p.inc(Message{Timestamp: time.Now(), Content: line, Level: LevelError})
	assert.Equal(t, LevelError, gotLevel)
	assert.Equal(t, line, gotMsg)
	assert.Nil(t, gotAttrs)
}

func TestParserCardinalityLimit(t *testing.T) {
	p := &Parser{
		patterns:              map[patternKey]*patternStat{},
		patternsPerLevel:      map[Level]int{},
		patternsPerLevelLimit: 2,
	}

	msgs := []string{
		"error alpha beta gamma",
		"error delta epsilon zeta",
		"error eta theta iota",
		"error kappa lambda mu",
	}
	for _, m := range msgs {
		p.inc(Message{Timestamp: time.Now(), Content: m, Level: LevelError})
	}
	assert.Equal(t, 2, p.patternsPerLevel[LevelError])

	fallbackKey := patternKey{level: LevelError, hash: unclassifiedPatternHash}
	stat, ok := p.patterns[fallbackKey]
	require.True(t, ok)
	assert.Equal(t, 2, stat.messages)
	assert.Equal(t, unclassifiedPatternLabel, stat.sample)

	counters := p.GetCounters()
	sort.Slice(counters, func(i, j int) bool { return counters[i].Sample < counters[j].Sample })

	assert.Equal(t, 3, len(counters))
	assert.Equal(t, msgs[0], counters[0].Sample)
	assert.Equal(t, msgs[1], counters[1].Sample)
	assert.Equal(t, unclassifiedPatternLabel, counters[2].Sample)
	assert.Equal(t, unclassifiedPatternHash, counters[2].Hash)
}

func TestParserRateLimit(t *testing.T) {
	calls := 0
	p := &Parser{
		patterns:              map[patternKey]*patternStat{},
		patternsPerLevel:      map[Level]int{},
		patternsPerLevelLimit: 256,
		limiter:               rate.NewLimiter(0, 3), // never refills
		onMsgCb: func(ts time.Time, level Level, patternHash string, msg string, attributes map[string]string) {
			calls++
		},
	}

	for i := 0; i < 10; i++ {
		p.inc(Message{Timestamp: time.Now(), Content: "error" + strings.Repeat(" word", i+1), Level: LevelError})
	}

	assert.Equal(t, 3, p.patternsPerLevel[LevelError])
	assert.Equal(t, 3, calls)

	stat, ok := p.patterns[patternKey{level: LevelError, hash: sampledPatternHash}]
	require.True(t, ok)
	assert.Equal(t, 7, stat.messages)
	assert.Equal(t, sampledPatternLabel, stat.sample)

	total := 0
	for _, c := range p.GetCounters() {
		assert.Equal(t, LevelError, c.Level)
		total += c.Messages
	}
	assert.Equal(t, 10, total)
}

func TestParserRateLimitLevels(t *testing.T) {
	p := &Parser{
		patterns:              map[patternKey]*patternStat{},
		patternsPerLevel:      map[Level]int{},
		patternsPerLevelLimit: 256,
		limiter:               rate.NewLimiter(0, 0), // always over the limit
	}

	p.inc(Message{Timestamp: time.Now(), Content: "some info message", Level: LevelInfo})
	p.inc(Message{Timestamp: time.Now(), Content: "some debug message", Level: LevelDebug})

	assert.Equal(t, 1, p.patterns[patternKey{level: LevelInfo, hash: ""}].messages)
	assert.Equal(t, 1, p.patterns[patternKey{level: LevelDebug, hash: ""}].messages)
	_, ok := p.patterns[patternKey{level: LevelError, hash: sampledPatternHash}]
	assert.False(t, ok)
}
