package logparser

import (
	"context"
	"sync"
	"time"
)

var (
	unclassifiedPatternLabel = "unclassified pattern (pattern limit reached)"
	unclassifiedPatternHash  = "00000000000000000000000000000000"
)

type LogEntry struct {
	Timestamp time.Time
	Content   string
	Level     Level
}

type LogCounter struct {
	Level    Level
	Hash     string
	Sample   string
	Messages int
}

type Parser struct {
	decoder Decoder

	patterns              map[patternKey]*patternStat
	patternsPerLevel      map[Level]int
	patternsPerLevelLimit int
	lock                  sync.RWMutex

	multilineCollector *MultilineCollector

	stop func()

	onMsgCb OnMsgCallbackF
}

type OnMsgCallbackF func(ts time.Time, level Level, patternHash string, msg string)

func NewParser(ch <-chan LogEntry, decoder Decoder, onMsgCallback OnMsgCallbackF, multilineCollectorTimeout time.Duration, patternsPerLevelLimit int) *Parser {
	p := &Parser{
		decoder:               decoder,
		patterns:              map[patternKey]*patternStat{},
		patternsPerLevel:      map[Level]int{},
		patternsPerLevelLimit: patternsPerLevelLimit,
		onMsgCb:               onMsgCallback,
	}
	ctx, stop := context.WithCancel(context.Background())
	p.stop = stop
	p.multilineCollector = NewMultilineCollector(ctx, multilineCollectorTimeout, multilineCollectorLimit)

	go func() {
		var err error
		for {
			select {
			case <-ctx.Done():
				return
			case entry := <-ch:
				if p.decoder != nil {
					if entry.Content, err = p.decoder.Decode(entry.Content); err != nil {
						continue
					}
				}
				p.multilineCollector.Add(entry)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-p.multilineCollector.Messages:
				p.inc(msg)
			}
		}
	}()

	return p
}

func (p *Parser) Stop() {
	p.stop()
}

func (p *Parser) inc(msg Message) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if msg.Level == LevelUnknown || msg.Level == LevelDebug || msg.Level == LevelInfo {
		key := patternKey{level: msg.Level, hash: ""}
		if stat := p.patterns[key]; stat == nil {
			p.patterns[key] = &patternStat{}
		}
		p.patterns[key].messages++
		if p.onMsgCb != nil {
			p.onMsgCb(msg.Timestamp, msg.Level, "", msg.Content)
		}
		return
	}

	pattern := NewPattern(msg.Content)
	stat, key := p.getPatternStat(msg.Level, pattern, msg.Content)
	if p.onMsgCb != nil {
		p.onMsgCb(msg.Timestamp, msg.Level, key.hash, msg.Content)
	}
	stat.messages++
}

func (p *Parser) getPatternStat(level Level, pattern *Pattern, sample string) (*patternStat, patternKey) {
	key := patternKey{level: level, hash: pattern.Hash()}
	if stat := p.patterns[key]; stat != nil {
		return stat, key
	}
	for k, ps := range p.patterns {
		if k.level != level || ps.pattern == nil {
			continue
		}
		if ps.pattern.WeakEqual(pattern) {
			return ps, k
		}
	}

	if p.patternsPerLevel[level] >= p.patternsPerLevelLimit {
		fallbackKey := patternKey{level: level, hash: unclassifiedPatternHash}
		stat := p.patterns[fallbackKey]
		if stat == nil {
			stat = &patternStat{sample: unclassifiedPatternLabel}
			p.patterns[fallbackKey] = stat
		}
		return stat, fallbackKey
	}

	stat := &patternStat{pattern: pattern, sample: sample}
	p.patterns[key] = stat
	p.patternsPerLevel[level]++
	return stat, key
}

func (p *Parser) GetCounters() []LogCounter {
	p.lock.RLock()
	defer p.lock.RUnlock()
	res := make([]LogCounter, 0, len(p.patterns))
	for k, ps := range p.patterns {
		res = append(res, LogCounter{Level: k.level, Hash: k.hash, Sample: ps.sample, Messages: ps.messages})
	}
	return res
}

type patternKey struct {
	level Level
	hash  string
}

type patternStat struct {
	pattern  *Pattern
	sample   string
	messages int
}
