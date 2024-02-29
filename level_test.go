package logparser

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGuessLevelGlog(t *testing.T) {
	//glog & klog
	assert.Equal(t, LevelUnknown, GuessLevel(`11002 a msg`))
	assert.Equal(t, LevelUnknown, GuessLevel(`WHAT1 a msg`))
	assert.Equal(t, LevelInfo, GuessLevel(`I0430 11:58:31.792717       1 cluster.go:337] memberlist 2020/04/30 11:58:31 [DEBUG] memberlist: Initiating push/pull sync with: 127.0.0.1:4000`))
	assert.Equal(t, LevelWarning, GuessLevel(`W0430 11:29:23.177635       1 nanny.go:120] Got EOF from stdout`))
	assert.Equal(t, LevelError, GuessLevel(`E0504 07:38:36.184861       1 replica_set.go:450] Sync "monitoring/prometheus-operator-5cfbdc9b67" failed with pods "prometheus-operator-5cfbdc9b67-" is forbidden: error looking up service account monitoring/prometheus-operator: serviceaccount "prometheus-operator" not found`))
	assert.Equal(t, LevelCritical, GuessLevel(`F0825 185142 test.cc:22] Check failed: write(1, NULL, 2) >= 0 Write NULL failed: Bad address [14]`))
}

func TestGuessLevelRedis(t *testing.T) {
	assert.Equal(t, LevelWarning, GuessLevel(`[4018] 14 Nov 07:01:22.119 * Background saving terminated with success`))
	assert.Equal(t, LevelInfo, GuessLevel(`1:S 12 Nov 07:52:11.999 - some msg`))
	assert.Equal(t, LevelDebug, GuessLevel(`1:S 12 Nov 2019 07:52:11.999 . verbosed`))
}

func TestGuessLevel(t *testing.T) {
	assert.Equal(t, LevelError, GuessLevel(`[Sat Dec 04 04:51:18 2020] [error] mod_jk child workerEnv in error state 6`))
	assert.Equal(t, LevelInfo, GuessLevel(`[info:2016-02-16T16:04:05.930-08:00] Some log text here`))
	assert.Equal(t, LevelInfo, GuessLevel(`2016-02-04T06:51:03.053580605Z" level=info msg="GET /containers/json`))
	assert.Equal(t, LevelError, GuessLevel(`2016-02-04T07:53:57.505612354Z" level=error msg="HTTP Error" err="No such image: -f" statusCode=404`))
	assert.Equal(t, LevelDebug, GuessLevel(`[2020-06-25 17:35:37,609][DEBUG][action.search            ] [srv] [tweets-100][6]`))
	assert.Equal(t, LevelError, GuessLevel(`[2023-10-12T09:56:53.393595+00:00] otel-php.ERROR: Export failure {"exception":"[object] (RuntimeException(code: 0): Export retry limit exceeded at /var/www/vendor/open-telemetry/sdk/Common/Export/Http/PsrTransport.php:114)","source":"OpenTelemetry\\Contrib\\Otlp\\SpanExporter"} []`))
	assert.Equal(t, LevelWarning, GuessLevel(`2023.10.12 13:58:41.168802 [ 847 ] {} <Warning> TCPHandler: Using deprecated interserver protocol because the client is too old. Consider upgrading all nodes in cluster.`))

	assert.Equal(t, LevelDebug, GuessLevel("[06:23:18 DBG] message"))
	assert.Equal(t, LevelInfo, GuessLevel("[06:23:18 INF] message"))
	assert.Equal(t, LevelWarning, GuessLevel("[06:23:18 WRN] message"))
	assert.Equal(t, LevelError, GuessLevel("[06:23:18 ERR] message"))
	assert.Equal(t, LevelCritical, GuessLevel("[06:23:18 FTL] message"))

	assert.Equal(t, LevelCritical, GuessLevel(`2024/02/29 11:01:03 [emerg] 1#1: duplicate location "/loc-path" in /etc/nginx/conf.d/default.conf:33`))
	assert.Equal(t, LevelCritical, GuessLevel(`nginx: [alert] could not open error log file: open() "/var/log/nginx/error.log" failed (13: Permission denied)`))
	assert.Equal(t, LevelCritical, GuessLevel(`2022/05/14 07:08:37 [crit] 6689#6689: *16721837 SSL_do_handshake() failed (SSL: error:1420918C:SSL routines:tls_early_post_process_client_hello:version too low) while SSL handshaking`))
	assert.Equal(t, LevelError, GuessLevel(`2009/01/01 19:45:44 [error]  29874#0: *98 open() "/var/www/one/nonexistent.html" failed (2: No such file or directory), client: 11.22.33.44, server: one.org, request: "GET /nonexistent.html HTTP/1.1", host: "one.org"`))
}
