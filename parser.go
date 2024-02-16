package logparser

import (
	"context"
	"sync"
	"time"
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

	patterns map[patternKey]*patternStat
	lock     sync.RWMutex

	multilineCollector *MultilineCollector

	stop func()

	onMsgCb OnMsgCallbackF
}

type OnMsgCallbackF func(ts time.Time, level Level, patternHash string, msg string)

func NewParser(ch <-chan LogEntry, decoder Decoder, onMsgCallback OnMsgCallbackF) *Parser {
	p := &Parser{
		decoder:  decoder,
		patterns: map[patternKey]*patternStat{},
		onMsgCb:  onMsgCallback,
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
	if p.onMsgCb != nil {
		p.onMsgCb(msg.Timestamp, msg.Level, key.hash, msg.Content)
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
