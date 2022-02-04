package logparser

import (
	"context"
	"strings"
	"sync"
	"time"
)

var (
	multilineCollectorTimeout = time.Millisecond * 100
)

type Message struct {
	Content string
	Level   Level
}

type MultilineCollector struct {
	Messages                     chan Message
	level                        Level
	lines                        []string
	isFirstLineContainsTimestamp bool
	timeout                      time.Duration
	lastReceiveTime              time.Time
	closed                       bool
	lock                         sync.Mutex
}

func NewMultilineCollector(ctx context.Context, timeout time.Duration) *MultilineCollector {
	m := &MultilineCollector{
		timeout:  timeout,
		Messages: make(chan Message, 1),
	}
	go m.dispatch(ctx)
	return m
}

func (m *MultilineCollector) dispatch(ctx context.Context) {
	ticker := time.NewTicker(m.timeout)
	defer ticker.Stop()
	defer close(m.Messages)

	for {
		select {
		case <-ctx.Done():
			m.closed = true
			return
		case t := <-ticker.C:
			m.lock.Lock()
			if t.Sub(m.lastReceiveTime) > m.timeout {
				m.flushMessage()
			}
			m.lock.Unlock()
		}
	}
}

func (m *MultilineCollector) Add(entry LogEntry) {
	m.lock.Lock()
	defer m.lock.Unlock()

	entry.Content = strings.TrimSuffix(entry.Content, "\n")
	if entry.Content == "" {
		if len(m.lines) > 0 {
			m.add(entry)
		}
		return
	}
	if !m.isPrevMsgPart(entry.Content) {
		m.flushMessage()
	}
	m.add(entry)
}

func (m *MultilineCollector) add(entry LogEntry) {
	if len(m.lines) == 0 {
		m.level = GuessLevel(entry.Content)
		if m.level == LevelUnknown && entry.Level != LevelUnknown {
			m.level = entry.Level
		}
		m.isFirstLineContainsTimestamp = containsTimestamp(entry.Content)
	}
	switch m.level {
	case LevelCritical, LevelError, LevelWarning:
		if len(m.lines) == 0 {
			m.appendLine(clean(entry.Content), entry.Content)
		} else {
			m.appendLine(entry.Content, entry.Content)
		}
	default:
		m.appendLine("", entry.Content)
	}
	m.lastReceiveTime = time.Now()
}

func (m *MultilineCollector) appendLine(value, rawValue string) {
	m.lines = append(m.lines, value)
}

func (m *MultilineCollector) isPrevMsgPart(l string) bool {
	if len(m.lines) == 0 {
		return true
	}

	if m.isFirstLineContainsTimestamp {
		if !containsTimestamp(l) {
			return true
		}
		return false
	}

	if strings.HasPrefix(l, "\tat ") || strings.HasPrefix(l, "\t... ") {
		return true
	}

	if strings.HasPrefix(l, "Traceback (") || strings.HasPrefix(l, "  File") || strings.HasPrefix(l, "    ") {
		return true
	}

	if strings.HasPrefix(l, "Caused by: ") {
		return true
	}
	if l == "The above exception was the direct cause of the following exception:" || l == "During handling of the above exception, another exception occurred:" {
		return true
	}
	prevLine := m.lines[len(m.lines)-1]
	if strings.HasPrefix(prevLine, "    ") || strings.HasPrefix(prevLine, "  File") || strings.HasSuffix(prevLine, "with root cause") {
		return true
	}
	return false
}

func (m *MultilineCollector) flushMessage() {
	if m.closed {
		return
	}
	if len(m.lines) == 0 {
		return
	}
	content := strings.TrimSpace(strings.Join(m.lines, "\n"))
	msg := Message{
		Content: content,
		Level:   m.level,
	}
	m.reset()
	m.Messages <- msg
}

func (m *MultilineCollector) reset() {
	m.lines = m.lines[:0]
	m.isFirstLineContainsTimestamp = false
	m.level = LevelUnknown
}
