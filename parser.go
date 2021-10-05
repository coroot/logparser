package logparser

import (
	"context"
	"sync"
)

type LogEntry struct {
	Content string
	Level   Level
}

type LogCounter struct {
	Level    Level
	Hash     string
	Sample   string
	Messages int
}

type Parser struct {
	decoder Decoder

	patterns map[patternKey]*patternStat
	lock     sync.RWMutex

	multilineCollector *MultilineCollector

	stop func()
}

func NewParser(ch <-chan LogEntry, decoder Decoder) *Parser {
	p := &Parser{decoder: decoder, patterns: map[patternKey]*patternStat{}}
	ctx, stop := context.WithCancel(context.Background())
	p.stop = stop
	p.multilineCollector = NewMultilineCollector(ctx, multilineCollectorTimeout)

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
		return
	}

	pattern := NewPattern(msg.Content)
	key := patternKey{level: msg.Level, hash: pattern.Hash()}
	stat := p.patterns[key]
	if stat == nil {
		for k, ps := range p.patterns {
			if k.level == msg.Level && ps.pattern.WeakEqual(pattern) {
				stat = ps
				break
			}
		}
		if stat == nil {
			stat = &patternStat{pattern: pattern, sample: msg.Content}
			p.patterns[key] = stat
		}
	}
	stat.messages++
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
