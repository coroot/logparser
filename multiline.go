package logparser

import (
	"context"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

var (
	multilineCollectorTimeout = time.Millisecond * 100
	multilineCollectorLimit   = 64 * 1024
)

type Message struct {
	Timestamp time.Time
	Content   string
	Level     Level
}

type MultilineCollector struct {
	Messages chan Message

	timeout time.Duration
	limit   int

	ts    time.Time
	level Level
	lines []string
	size  int

	lock            sync.Mutex
	closed          bool
	lastReceiveTime time.Time

	isFirstLineContainsTimestamp bool
	pythonTraceback              bool
	pythonTracebackExpected      bool
}

func NewMultilineCollector(ctx context.Context, timeout time.Duration, limit int) *MultilineCollector {
	m := &MultilineCollector{
		timeout:  timeout,
		limit:    limit,
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
	if !utf8.ValidString(entry.Content) {
		return
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	entry.Content = strings.TrimSuffix(entry.Content, "\n")
	if entry.Content == "" {
		if len(m.lines) > 0 {
			m.add(entry)
		}
		return
	}
	if m.isNextMessage(entry.Content) {
		pythonTraceback := m.pythonTraceback
		m.flushMessage()
		m.pythonTraceback = pythonTraceback
	}
	m.add(entry)
}

func (m *MultilineCollector) add(entry LogEntry) {
	remaining := m.limit - m.size
	if remaining <= 0 {
		return
	}
	if len(m.lines) == 0 {
		m.ts = entry.Timestamp
		m.level = GuessLevel(entry.Content)
		if m.level == LevelUnknown && entry.Level != LevelUnknown {
			m.level = entry.Level
		}
		m.isFirstLineContainsTimestamp = containsTimestamp(entry.Content)
	}
	content := entry.Content
	if len(content) > remaining {
		content = content[:remaining]
	}
	m.lines = append(m.lines, content)
	m.size += len(content) + 1
	m.lastReceiveTime = time.Now()
}

func (m *MultilineCollector) isNextMessage(l string) bool {
	if l == "" || l == "}" || strings.HasPrefix(l, "\t") || strings.HasPrefix(l, "  ") {
		return false
	}

	if m.isFirstLineContainsTimestamp {
		return containsTimestamp(l)
	}

	if strings.HasPrefix(l, "Caused by: ") {
		return false
	}

	if strings.HasPrefix(l, "for call at") {
		return false
	}

	if strings.HasPrefix(l, "Traceback ") {
		m.pythonTraceback = true
		if m.pythonTracebackExpected {
			m.pythonTracebackExpected = false
			return false
		}
		return len(m.lines) > 0
	}
	if l == "The above exception was the direct cause of the following exception:" || l == "During handling of the above exception, another exception occurred:" {
		m.pythonTracebackExpected = true
		return false
	}
	if m.pythonTraceback {
		m.pythonTraceback = false
		return false
	}

	return true
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
		Timestamp: m.ts,
		Content:   content,
		Level:     m.level,
	}
	m.reset()
	m.Messages <- msg
}

func (m *MultilineCollector) reset() {
	m.ts = time.Time{}
	m.level = LevelUnknown
	m.lines = m.lines[:0]
	m.size = 0
	m.isFirstLineContainsTimestamp = false
	m.pythonTraceback = false
	m.pythonTracebackExpected = false
}
