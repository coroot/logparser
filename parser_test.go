package logparser

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	}
	for _, m := range msgs {
		p.inc(Message{Timestamp: time.Now(), Content: m, Level: LevelError})
	}
	assert.Equal(t, 2, p.patternsPerLevel[LevelError])

	fallbackKey := patternKey{level: LevelError, hash: ""}
	stat, ok := p.patterns[fallbackKey]
	require.True(t, ok)
	assert.Equal(t, 1, stat.messages)
	assert.Equal(t, unclassifiedPatternLabel, stat.sample)

	counters := p.GetCounters()
	sort.Slice(counters, func(i, j int) bool { return counters[i].Sample < counters[j].Sample })

	assert.Equal(t, 3, len(counters))
	assert.Equal(t, msgs[0], counters[0].Sample)
	assert.Equal(t, msgs[1], counters[1].Sample)
	assert.Equal(t, unclassifiedPatternLabel, counters[2].Sample)
}
