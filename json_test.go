package logparser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseJsonLogNotJson(t *testing.T) {
	for _, s := range []string{
		"",
		"plain text message",
		"{incomplete json",
		`{"a":1} trailing garbage`,
		`{"a":1}{"b":2}`,
		"[1, 2, 3]",
		"{}",
		"ts=2024-01-01 level=info msg=logfmt",
	} {
		assert.Nil(t, ParseJsonLog(s), s)
	}
}

func TestParseJsonLogSlog(t *testing.T) {
	l := ParseJsonLog(`{"time":"2026-07-24T15:49:24.699496383Z","level":"WARN","msg":"retrying request","upstream":"payment-service","attempt":3,"error":"connection refused"}`)
	require.NotNil(t, l)
	assert.Equal(t, LevelWarning, l.Level)
	assert.Equal(t, "retrying request", l.Message)
	assert.Equal(t, map[string]string{
		"upstream": "payment-service",
		"attempt":  "3",
		"error":    "connection refused",
	}, l.Attributes)
}

func TestParseJsonLogZap(t *testing.T) {
	l := ParseJsonLog(`{"level":"error","ts":1784908164.831705,"caller":"app/main.go:159","msg":"request to upstream failed","upstream":"inventory-service","error":"context deadline exceeded"}`)
	require.NotNil(t, l)
	assert.Equal(t, LevelError, l.Level)
	assert.Equal(t, "request to upstream failed", l.Message)
	assert.Equal(t, map[string]string{
		"caller":   "app/main.go:159",
		"upstream": "inventory-service",
		"error":    "context deadline exceeded",
	}, l.Attributes)
}

func TestParseJsonLogLogrus(t *testing.T) {
	l := ParseJsonLog(`{"amount":181.72,"currency":"USD","level":"info","msg":"payment processed","order_id":"ord-753339","time":"2026-07-24T15:49:24Z"}`)
	require.NotNil(t, l)
	assert.Equal(t, LevelInfo, l.Level)
	assert.Equal(t, "payment processed", l.Message)
	assert.Equal(t, map[string]string{
		"amount":   "181.72",
		"currency": "USD",
		"order_id": "ord-753339",
	}, l.Attributes)
}

func TestParseJsonLogDotnet(t *testing.T) {
	l := ParseJsonLog(`{"Timestamp":"2026-07-24T15:49:25.1236791Z","EventId":0,"LogLevel":"Information","Category":"Demo.Worker","Message":"request completed","State":{"Message":"request completed","status":200,"{OriginalFormat}":"request completed"}}`)
	require.NotNil(t, l)
	assert.Equal(t, LevelInfo, l.Level)
	assert.Equal(t, "request completed", l.Message)
	assert.Equal(t, map[string]string{
		"EventId":                "0",
		"Category":               "Demo.Worker",
		"State.Message":          "request completed",
		"State.status":           "200",
		"State.{OriginalFormat}": "request completed",
	}, l.Attributes)
}

func TestParseJsonLogSerilogClef(t *testing.T) {
	l := ParseJsonLog(`{"@t":"2026-07-24T15:49:25.2750856Z","@mt":"slow query","@l":"Warning","duration_ms":2026.79,"rows":7552}`)
	require.NotNil(t, l)
	assert.Equal(t, LevelWarning, l.Level)
	assert.Equal(t, "slow query", l.Message)
	assert.Equal(t, map[string]string{
		"duration_ms": "2026.79",
		"rows":        "7552",
	}, l.Attributes)

	l = ParseJsonLog(`{"@t":"2026-07-24T15:49:25.2634236Z","@mt":"request completed","status":200}`)
	require.NotNil(t, l)
	assert.Equal(t, LevelUnknown, l.Level)
	assert.Equal(t, "request completed", l.Message)
}

func TestParseJsonLogPino(t *testing.T) {
	l := ParseJsonLog(`{"level":30,"time":1784908164831,"pid":1,"hostname":"review-service-775df668b6-abcde","msg":"request completed","responseTime":12}`)
	require.NotNil(t, l)
	assert.Equal(t, LevelInfo, l.Level)
	assert.Equal(t, "request completed", l.Message)
	assert.Equal(t, "12", l.Attributes["responseTime"])

	l = ParseJsonLog(`{"level":50,"time":1784908164831,"msg":"unhandled error"}`)
	require.NotNil(t, l)
	assert.Equal(t, LevelError, l.Level)
}

func TestParseJsonLogKeyPriority(t *testing.T) {
	l := ParseJsonLog(`{"log":"b","msg":"a","message":"c","level":"warn","severity":"error"}`)
	require.NotNil(t, l)
	assert.Equal(t, "a", l.Message)
	assert.Equal(t, LevelWarning, l.Level)
	assert.Equal(t, map[string]string{
		"log":      "b",
		"message":  "c",
		"severity": "error",
	}, l.Attributes)

	l = ParseJsonLog(`{"msg":42,"message":"m"}`)
	require.NotNil(t, l)
	assert.Equal(t, "m", l.Message)
	assert.Equal(t, "42", l.Attributes["msg"])

	l = ParseJsonLog(`{"msg":"m","level":"weird","severity":"error"}`)
	require.NotNil(t, l)
	assert.Equal(t, LevelError, l.Level)
	assert.Equal(t, "weird", l.Attributes["level"])
}

func TestParseJsonLogNestedAndArrays(t *testing.T) {
	l := ParseJsonLog(`{"msg":"m","http":{"method":"GET","response":{"status":200}},"tags":["a","b"],"ok":true,"ref":null}`)
	require.NotNil(t, l)
	assert.Equal(t, map[string]string{
		"http.method":          "GET",
		"http.response.status": "200",
		"tags":                 `["a","b"]`,
		"ok":                   "true",
		"ref":                  "",
	}, l.Attributes)
}

func TestParseJsonLogAttrLimits(t *testing.T) {
	sb := strings.Builder{}
	sb.WriteString(`{"msg":"m"`)
	for i := 0; i < 500; i++ {
		fmt.Fprintf(&sb, `,"key%d":"v"`, i)
	}
	sb.WriteString("}")
	l := ParseJsonLog(sb.String())
	require.NotNil(t, l)
	assert.LessOrEqual(t, len(l.Attributes), jsonMaxAttrs)

	l = ParseJsonLog(`{"msg":"m","big":"` + strings.Repeat("a", 10000) + `"}`)
	require.NotNil(t, l)
	assert.Equal(t, jsonMaxValueLen, len(l.Attributes["big"]))
}

func TestParseJsonLogLevels(t *testing.T) {
	assert.Equal(t, LevelDebug, jsonLevelFromString("TRACE"))
	assert.Equal(t, LevelDebug, jsonLevelFromString("Verbose"))
	assert.Equal(t, LevelInfo, jsonLevelFromString("Information"))
	assert.Equal(t, LevelInfo, jsonLevelFromString("notice"))
	assert.Equal(t, LevelWarning, jsonLevelFromString("WARN"))
	assert.Equal(t, LevelError, jsonLevelFromString("Error"))
	assert.Equal(t, LevelCritical, jsonLevelFromString("panic"))
	assert.Equal(t, LevelCritical, jsonLevelFromString("FATAL"))
	assert.Equal(t, LevelUnknown, jsonLevelFromString("something else"))
}

var (
	benchSlogLine   = `{"time":"2026-07-24T15:49:24.699496383Z","level":"WARN","msg":"retrying request","upstream":"payment-service","attempt":3,"error":"dial tcp 10.96.14.21:8080: connect: connection refused"}`
	benchPlainLine  = `2026-07-24 15:49:24.699 WARN [main] retrying request to payment-service, attempt 3: connection refused`
	benchBracedLine = "{this line starts with a brace but is not json, like a C++ initializer dump or a truncated record" + strings.Repeat(" x", 30) + "}"
	benchNestedLine = `{"time":"2026-07-24T15:49:24.699496383Z","level":"info","msg":"request completed","http":{"request":{"method":"POST","path":"/api/checkout","headers":{"user-agent":"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36","x-request-id":"e2b1a8c0-6f4e-4b6a-9a3e-2c8d1f0b7a55"}},"response":{"status":200,"duration_ms":138.04,"bytes":2048}},"db":{"queries":12,"duration_ms":41.7},"user":{"id":"user-1453","tenant":"acme","roles":["admin","billing"]},"cache":{"hits":9,"misses":1}}`
)

func BenchmarkParseJsonLogSlog(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if ParseJsonLog(benchSlogLine) == nil {
			b.Fatal("expected json")
		}
	}
}

func BenchmarkParseJsonLogNested(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if ParseJsonLog(benchNestedLine) == nil {
			b.Fatal("expected json")
		}
	}
}

func BenchmarkParseJsonLogPlainText(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if ParseJsonLog(benchPlainLine) != nil {
			b.Fatal("expected non-json")
		}
	}
}

func BenchmarkParseJsonLogBracedNonJson(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if ParseJsonLog(benchBracedLine) != nil {
			b.Fatal("expected non-json")
		}
	}
}
