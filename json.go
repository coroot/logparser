package logparser

import (
	"encoding/json"
	"io"
	"strings"
	"unicode/utf8"
)

const (
	jsonMaxAttrs    = 100
	jsonMaxDepth    = 8
	jsonMaxValueLen = 4096
)

var (
	jsonMessageKeys = map[string]int{
		"msg":     0,
		"message": 1,
		"@m":      2,
		"@mt":     3,
		"body":    4,
		"log":     5,
	}
	jsonLevelKeys = map[string]int{
		"level":         0,
		"severity":      1,
		"loglevel":      2,
		"log.level":     3,
		"lvl":           4,
		"@l":            5,
		"severity_text": 6,
		"levelname":     7,
	}
	jsonTimestampKeys = map[string]bool{
		"time": true, "timestamp": true, "ts": true, "@t": true, "@timestamp": true, "datetime": true, "asctime": true,
	}
)

type JsonLog struct {
	Level      Level
	Message    string
	Attributes map[string]string
}

func ParseJsonLog(content string) *JsonLog {
	if len(content) < 2 || content[0] != '{' || content[len(content)-1] != '}' {
		return nil
	}
	d := json.NewDecoder(strings.NewReader(content))
	d.UseNumber()
	var fields map[string]any
	if d.Decode(&fields) != nil || len(fields) == 0 {
		return nil
	}
	if _, err := d.Token(); err != io.EOF {
		return nil
	}

	res := &JsonLog{Attributes: map[string]string{}}
	var msgKey, lvlKey string
	var msgPrio, lvlPrio int
	var msgVal string
	var lvlRaw any
	for k, v := range fields {
		lk := strings.ToLower(k)
		if p, ok := jsonMessageKeys[lk]; ok {
			if s, ok := v.(string); ok {
				if msgKey == "" || p < msgPrio || (p == msgPrio && k < msgKey) {
					if msgKey != "" {
						flattenJsonField(msgKey, msgVal, res.Attributes, jsonMaxDepth)
					}
					msgKey, msgPrio, msgVal = k, p, s
				} else {
					flattenJsonField(k, v, res.Attributes, jsonMaxDepth)
				}
				continue
			}
		} else if p, ok := jsonLevelKeys[lk]; ok {
			if lvl := jsonLevelFromValue(v); lvl != LevelUnknown {
				if lvlKey == "" || p < lvlPrio || (p == lvlPrio && k < lvlKey) {
					if lvlKey != "" {
						flattenJsonField(lvlKey, lvlRaw, res.Attributes, jsonMaxDepth)
					}
					lvlKey, lvlPrio, lvlRaw = k, p, v
					res.Level = lvl
				} else {
					flattenJsonField(k, v, res.Attributes, jsonMaxDepth)
				}
				continue
			}
		} else if jsonTimestampKeys[lk] {
			switch v.(type) {
			case string, json.Number:
				continue
			}
		}
		flattenJsonField(k, v, res.Attributes, jsonMaxDepth)
	}
	res.Message = strings.TrimSuffix(msgVal, "\n")
	return res
}

func jsonLevelFromValue(v any) Level {
	switch value := v.(type) {
	case string:
		return jsonLevelFromString(value)
	case json.Number:
		return jsonLevelFromNumber(value)
	}
	return LevelUnknown
}

func flattenJsonField(key string, value any, attrs map[string]string, depth int) {
	if len(attrs) >= jsonMaxAttrs {
		return
	}
	switch v := value.(type) {
	case nil:
		attrs[key] = ""
	case string:
		attrs[key] = truncateUtf8(v, jsonMaxValueLen)
	case bool:
		if v {
			attrs[key] = "true"
		} else {
			attrs[key] = "false"
		}
	case json.Number:
		attrs[key] = v.String()
	case map[string]any:
		if depth <= 0 {
			attrs[key] = jsonEncode(v)
			return
		}
		for k, vv := range v {
			flattenJsonField(key+"."+k, vv, attrs, depth-1)
		}
	default:
		attrs[key] = jsonEncode(v)
	}
}

func jsonEncode(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return truncateUtf8(string(data), jsonMaxValueLen)
}

func truncateUtf8(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	for limit > 0 && !utf8.RuneStart(s[limit]) {
		limit--
	}
	return s[:limit]
}

func jsonLevelFromString(s string) Level {
	if len(s) > 16 {
		return LevelUnknown
	}
	switch strings.ToLower(s) {
	case "trace", "trc", "debug", "dbg", "verbose":
		return LevelDebug
	case "info", "inf", "information", "informational", "notice":
		return LevelInfo
	case "warn", "wrn", "warning":
		return LevelWarning
	case "error", "err":
		return LevelError
	case "fatal", "ftl", "critical", "crit", "panic", "dpanic", "alert", "emerg", "emergency":
		return LevelCritical
	}
	return LevelUnknown
}

func jsonLevelFromNumber(n json.Number) Level {
	v, err := n.Int64()
	if err != nil {
		return LevelUnknown
	}
	switch {
	case v < 10:
		return LevelUnknown
	case v < 30:
		return LevelDebug
	case v < 40:
		return LevelInfo
	case v < 50:
		return LevelWarning
	case v < 60:
		return LevelError
	case v < 100:
		return LevelCritical
	}
	return LevelUnknown
}
