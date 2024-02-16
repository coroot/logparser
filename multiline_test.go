package logparser

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func writeByLine(m *MultilineCollector, data string, ts time.Time) []Message {
	var msgs []Message
	done := make(chan bool)
	go func() {
		timer := time.NewTimer(3 * m.timeout)
		for {
			select {
			case <-timer.C:
				done <- true
				return
			case msg := <-m.Messages:
				msgs = append(msgs, msg)
			}
		}
	}()
	for _, line := range strings.Split(data, "\n") {
		m.Add(LogEntry{Timestamp: ts, Content: line, Level: LevelUnknown})
		ts = ts.Add(time.Millisecond)
	}
	<-done
	return msgs
}

func TestMultilineCollector(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	m := NewMultilineCollector(ctx, 10*time.Millisecond, multilineCollectorLimit)
	defer cancel()

	data := `Order response: {"statusCode":406,"body":{"timestamp":1648205755430,"status":406,"error":"Not Acceptable","exception":"works.weave.socks.orders.controllers.OrdersController$PaymentDeclinedException","message":"Payment declined: amount exceeds 100.00","path":"/orders"},"headers":{"x-application-context":"orders:80","content-type":"application/json;charset=UTF-8","transfer-encoding":"chunked","date":"Fri, 25 Mar 2022 10:55:55 GMT","connection":"close"},"request":{"uri":{"protocol":"http:","slashes":true,"auth":null,"host":"orders","port":80,"hostname":"orders","hash":null,"search":null,"query":null,"pathname":"/orders","path":"/orders","href":"http://orders/orders"},"method":"POST","headers":{"accept":"application/json","content-type":"application/json","content-length":232}}}
Order response: {"timestamp":1648205755430,"status":406,"error":"Not Acceptable","exception":"works.weave.socks.orders.controllers.OrdersController$PaymentDeclinedException","message":"Payment declined: amount exceeds 100.00","path":"/orders"}`
	msgs := writeByLine(m, data, time.Unix(0, 0))
	require.Len(t, msgs, 2)
	assert.Equal(t, strings.Split(data, "\n")[0], msgs[0].Content)
	assert.Equal(t, strings.Split(data, "\n")[1], msgs[1].Content)
}

func TestMultilineCollectorPython(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	m := NewMultilineCollector(ctx, 10*time.Millisecond, multilineCollectorLimit)
	defer cancel()

	data := `Traceback (most recent call last):
  File "/Users/user/workspace/pythonProject/main.py", line 10, in <module>
    func()
  File "/Users/user/workspace/pythonProject/main.py", line 4, in func
    raise ConnectionError
ConnectionError`
	msgs := writeByLine(m, data, time.Unix(0, 0))
	require.Len(t, msgs, 1)
	assert.Equal(t, data, msgs[0].Content)

	data = `Traceback (most recent call last):
  File "/Users/user/workspace/pythonProject/main.py", line 10, in <module>
    func()
  File "/Users/user/workspace/pythonProject/main.py", line 4, in func
    raise ConnectionError
ConnectionError
Traceback (most recent call last):
  File "/Users/user/workspace/pythonProject/main.py", line 10, in <module>
    func()
  File "/Users/user/workspace/pythonProject/main.py", line 4, in func
    raise ConnectionError
ConnectionError`
	msgs = writeByLine(m, data, time.Unix(0, 0))
	require.Len(t, msgs, 2)
	assert.Equal(t, data, msgs[0].Content+"\n"+msgs[0].Content)

	data = `Traceback (most recent call last):
  File "/Users/user/workspace/pythonProject/main.py", line 10, in <module>
    func()
  File "/Users/user/workspace/pythonProject/main.py", line 4, in func
    raise ConnectionError
ConnectionError

The above exception was the direct cause of the following exception:

Traceback (most recent call last):
  File "/Users/user/workspace/pythonProject/main.py", line 12, in <module>
    raise RuntimeError('Failed to open database') from exc
RuntimeError: Failed to open database`
	msgs = writeByLine(m, data, time.Unix(0, 0))
	require.Len(t, msgs, 1)
	assert.Equal(t, data, msgs[0].Content)

	data = `Traceback (most recent call last):
  File "/Users/user/workspace/pythonProject/main.py", line 10, in <module>
    func()
  File "/Users/user/workspace/pythonProject/main.py", line 4, in func
    raise ConnectionError
ConnectionError

The above exception was the direct cause of the following exception:

Traceback (most recent call last):
  File "/Users/user/workspace/pythonProject/main.py", line 12, in <module>
    raise RuntimeError('Failed to open database') from exc
RuntimeError: Failed to open database

During handling of the above exception, another exception occurred:

Traceback (most recent call last):
  File "/Users/user/workspace/pythonProject/main.py", line 14, in <module>
    raise ConnectionError
ConnectionError`
	msgs = writeByLine(m, data, time.Unix(0, 0))
	require.Len(t, msgs, 1)
	assert.Equal(t, data, msgs[0].Content)

	data = `2020-03-20 08:48:57,067 ERROR [django.request:222] log 46 140452532862280 Internal Server Error: /article
Traceback (most recent call last):
  File "/usr/local/lib/python3.8/site-packages/django/db/backends/base/base.py", line 220, in ensure_connection
    self.connect()
  File "/usr/local/lib/python3.8/site-packages/django/utils/asyncio.py", line 26, in inner
    return func(*args, **kwargs)
  File "/usr/local/lib/python3.8/site-packages/django/db/backends/base/base.py", line 197, in connect
    self.connection = self.get_new_connection(conn_params)
  File "/usr/local/lib/python3.8/site-packages/django_prometheus/db/common.py", line 44, in get_new_connection
    return super(DatabaseWrapperMixin, self).get_new_connection(*args, **kwargs)
  File "/usr/local/lib/python3.8/site-packages/django/utils/asyncio.py", line 26, in inner
    return func(*args, **kwargs)
  File "/usr/local/lib/python3.8/site-packages/django/db/backends/mysql/base.py", line 233, in get_new_connection
    return Database.connect(**conn_params)
  File "/usr/local/lib/python3.8/site-packages/MySQLdb/__init__.py", line 84, in Connect
    return Connection(*args, **kwargs)
  File "/usr/local/lib/python3.8/site-packages/MySQLdb/connections.py", line 179, in __init__
    super(Connection, self).__init__(*args, **kwargs2)
MySQLdb._exceptions.OperationalError: (1040, 'Too many connections')

The above exception was the direct cause of the following exception:

Traceback (most recent call last):
  File "/usr/local/lib/python3.8/site-packages/django/core/handlers/exception.py", line 34, in inner
    response = get_response(request)
  File "/usr/local/lib/python3.8/site-packages/django/core/handlers/base.py", line 115, in _get_response
    response = self.process_exception_by_middleware(e, request)
  File "/usr/local/lib/python3.8/site-packages/django/core/handlers/base.py", line 113, in _get_response
    response = wrapped_callback(request, *callback_args, **callback_kwargs)
  File "/usr/local/lib/python3.8/contextlib.py", line 74, in inner
    with self._recreate_cm():
  File "/usr/local/lib/python3.8/site-packages/django/db/transaction.py", line 175, in __enter__
    if not connection.get_autocommit():
  File "/usr/local/lib/python3.8/site-packages/django/db/backends/base/base.py", line 390, in get_autocommit
    self.ensure_connection()
  File "/usr/local/lib/python3.8/site-packages/django/utils/asyncio.py", line 26, in inner
    return func(*args, **kwargs)
  File "/usr/local/lib/python3.8/site-packages/django/db/backends/base/base.py", line 220, in ensure_connection
    self.connect()
  File "/usr/local/lib/python3.8/site-packages/django/db/utils.py", line 90, in __exit__
    raise dj_exc_value.with_traceback(traceback) from exc_value
  File "/usr/local/lib/python3.8/site-packages/django/db/backends/base/base.py", line 220, in ensure_connection
    self.connect()
  File "/usr/local/lib/python3.8/site-packages/django/utils/asyncio.py", line 26, in inner
    return func(*args, **kwargs)"
  File "/usr/local/lib/python3.8/site-packages/django/db/backends/base/base.py", line 197, in connect
    self.connection = self.get_new_connection(conn_params)
  File "/usr/local/lib/python3.8/site-packages/django_prometheus/db/common.py", line 44, in get_new_connection
    return super(DatabaseWrapperMixin, self).get_new_connection(*args, **kwargs)
  File "/usr/local/lib/python3.8/site-packages/django/utils/asyncio.py", line 26, in inner
    return func(*args, **kwargs)
  File "/usr/local/lib/python3.8/site-packages/django/db/backends/mysql/base.py", line 233, in get_new_connection
    return Database.connect(**conn_params)
  File "/usr/local/lib/python3.8/site-packages/MySQLdb/__init__.py", line 84, in Connect
    return Connection(*args, **kwargs)
  File "/usr/local/lib/python3.8/site-packages/MySQLdb/connections.py", line 179, in __init__
    super(Connection, self).__init__(*args, **kwargs2)
django.db.utils.OperationalError: (1040, 'Too many connections')`

	msgs = writeByLine(m, data, time.Unix(100500, 0))
	require.Len(t, msgs, 1)
	assert.Equal(t, data, msgs[0].Content)
	assert.Equal(t, time.Unix(100500, 0), msgs[0].Timestamp)

	data = `2020-03-20 08:48:57,067 ERROR:__main__:Traceback (most recent call last):
  File "<stdin>", line 2, in <module>
  File "<stdin>", line 2, in do_something_that_might_error
  File "<stdin>", line 2, in raise_error
RuntimeError: something bad happened!`
	msgs = writeByLine(m, data, time.Unix(0, 0))
	require.Len(t, msgs, 1)
	assert.Equal(t, data, msgs[0].Content)
}

func TestMultilineCollectorJava(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	m := NewMultilineCollector(ctx, 10*time.Millisecond, multilineCollectorLimit)
	defer cancel()

	data := `Exception in thread "main" java.lang.NullPointerException
	at com.example.MyClass.methodA(MyClass.java:10)
	at com.example.MyClass.methodB(MyClass.java:20)
	at com.example.MyClass.main(MyClass.java:30)
Caused by: java.lang.ArrayIndexOutOfBoundsException: Index 5 out of bounds for length 5
	at com.example.AnotherClass.anotherMethod(AnotherClass.java:15)
	at com.example.MyClass.methodA(MyClass.java:8)
	... 2 more`
	msgs := writeByLine(m, data, time.Unix(0, 0))
	require.Len(t, msgs, 1)
	assert.Equal(t, data, msgs[0].Content)

	data = `Exception in thread "main" java.lang.NullPointerException
	at com.example.MyClass.methodA(MyClass.java:10)
	at com.example.MyClass.methodB(MyClass.java:20)
	at com.example.MyClass.main(MyClass.java:30)
Caused by: java.lang.ArrayIndexOutOfBoundsException: Index 5 out of bounds for length 5
	at com.example.AnotherClass.anotherMethod(AnotherClass.java:15)
	at com.example.MyClass.methodA(MyClass.java:8)
	... 2 more
Exception in thread "main" java.lang.NullPointerException
	at com.example.MyClass.methodA(MyClass.java:10)
	at com.example.MyClass.methodB(MyClass.java:20)
	at com.example.MyClass.main(MyClass.java:30)
Caused by: java.lang.ArrayIndexOutOfBoundsException: Index 5 out of bounds for length 5
	at com.example.AnotherClass.anotherMethod(AnotherClass.java:15)
	at com.example.MyClass.methodA(MyClass.java:8)
	... 2 more`
	msgs = writeByLine(m, data, time.Unix(0, 0))
	require.Len(t, msgs, 2)
	assert.Equal(t, data, msgs[0].Content+"\n"+msgs[1].Content)

	data = `ERROR [Messaging-EventLoop-3-1] 2023-10-04 14:27:35,249 2020-03-31 javax.servlet.ServletException: Something bad happened
	at com.example.myproject.OpenSessionInViewFilter.doFilter(OpenSessionInViewFilter.java:60)
	at org.mortbay.jetty.servlet.ServletHandler$CachedChain.doFilter(ServletHandler.java:1157)
	at com.example.myproject.ExceptionHandlerFilter.doFilter(ExceptionHandlerFilter.java:28)
	at org.mortbay.jetty.servlet.ServletHandler$CachedChain.doFilter(ServletHandler.java:1157)
	at com.example.myproject.OutputBufferFilter.doFilter(OutputBufferFilter.java:33)
	at org.mortbay.jetty.servlet.ServletHandler$CachedChain.doFilter(ServletHandler.java:1157)
	at org.mortbay.jetty.servlet.ServletHandler.handle(ServletHandler.java:388)
	at org.mortbay.jetty.security.SecurityHandler.handle(SecurityHandler.java:216)
	at org.mortbay.jetty.servlet.SessionHandler.handle(SessionHandler.java:182)
	at org.mortbay.jetty.handler.ContextHandler.handle(ContextHandler.java:765)
	at org.mortbay.jetty.webapp.WebAppContext.handle(WebAppContext.java:418)
	at org.mortbay.jetty.handler.HandlerWrapper.handle(HandlerWrapper.java:152)
	at org.mortbay.jetty.Server.handle(Server.java:326)
	at org.mortbay.jetty.HttpConnection.handleRequest(HttpConnection.java:542)
	at org.mortbay.jetty.HttpConnection$RequestHandler.content(HttpConnection.java:943)
	at org.mortbay.jetty.HttpParser.parseNext(HttpParser.java:756)
	at org.mortbay.jetty.HttpParser.parseAvailable(HttpParser.java:218)
	at org.mortbay.jetty.HttpConnection.handle(HttpConnection.java:404)
	at org.mortbay.jetty.bio.SocketConnector$Connection.run(SocketConnector.java:228)
	at org.mortbay.thread.QueuedThreadPool$PoolThread.run(QueuedThreadPool.java:582)
Caused by: com.example.myproject.MyProjectServletException
	at com.example.myproject.MyServlet.doPost(MyServlet.java:169)
	at javax.servlet.http.HttpServlet.service(HttpServlet.java:727)
	at javax.servlet.http.HttpServlet.service(HttpServlet.java:820)
	at org.mortbay.jetty.servlet.ServletHolder.handle(ServletHolder.java:511)
	at org.mortbay.jetty.servlet.ServletHandler$CachedChain.doFilter(ServletHandler.java:1166)
	at com.example.myproject.OpenSessionInViewFilter.doFilter(OpenSessionInViewFilter.java:30)
	... 27 more
Caused by: org.hibernate.exception.ConstraintViolationException: could not insert: [com.example.myproject.MyEntity]
	at org.hibernate.exception.SQLStateConverter.convert(SQLStateConverter.java:96)
	at org.hibernate.exception.JDBCExceptionHelper.convert(JDBCExceptionHelper.java:66)
	at org.hibernate.id.insert.AbstractSelectingDelegate.performInsert(AbstractSelectingDelegate.java:64)
	at org.hibernate.persister.entity.AbstractEntityPersister.insert(AbstractEntityPersister.java:2329)
	at org.hibernate.persister.entity.AbstractEntityPersister.insert(AbstractEntityPersister.java:2822)
	at org.hibernate.action.EntityIdentityInsertAction.execute(EntityIdentityInsertAction.java:71)
	at org.hibernate.engine.ActionQueue.execute(ActionQueue.java:268)
	at org.hibernate.event.def.AbstractSaveEventListener.performSaveOrReplicate(AbstractSaveEventListener.java:321)
	at org.hibernate.event.def.AbstractSaveEventListener.performSave(AbstractSaveEventListener.java:204)
	at org.hibernate.event.def.AbstractSaveEventListener.saveWithGeneratedId(AbstractSaveEventListener.java:130)
	at org.hibernate.event.def.DefaultSaveOrUpdateEventListener.saveWithGeneratedOrRequestedId(DefaultSaveOrUpdateEventListener.java:210)
	at org.hibernate.event.def.DefaultSaveEventListener.saveWithGeneratedOrRequestedId(DefaultSaveEventListener.java:56)
	at org.hibernate.event.def.DefaultSaveOrUpdateEventListener.entityIsTransient(DefaultSaveOrUpdateEventListener.java:195)
	at org.hibernate.event.def.DefaultSaveEventListener.performSaveOrUpdate(DefaultSaveEventListener.java:50)
	at org.hibernate.event.def.DefaultSaveOrUpdateEventListener.onSaveOrUpdate(DefaultSaveOrUpdateEventListener.java:93)
	at org.hibernate.impl.SessionImpl.fireSave(SessionImpl.java:705)
	at org.hibernate.impl.SessionImpl.save(SessionImpl.java:693)
	at org.hibernate.impl.SessionImpl.save(SessionImpl.java:689)
	at sun.reflect.GeneratedMethodAccessor5.invoke(Unknown Source)
	at sun.reflect.DelegatingMethodAccessorImpl.invoke(DelegatingMethodAccessorImpl.java:25)
	at java.lang.reflect.Method.invoke(Method.java:597)
	at org.hibernate.context.ThreadLocalSessionContext$TransactionProtectionWrapper.invoke(ThreadLocalSessionContext.java:344)
	at $Proxy19.save(Unknown Source)
	at com.example.myproject.MyEntityService.save(MyEntityService.java:59) <-- relevant call (see notes below)
	at com.example.myproject.MyServlet.doPost(MyServlet.java:164)
	... 32 more
Caused by: java.sql.SQLException: Violation of unique constraint MY_ENTITY_UK_1: duplicate value(s) for column(s) MY_COLUMN in statement [...]
	at org.hsqldb.jdbc.Util.throwError(Unknown Source)
	at org.hsqldb.jdbc.jdbcPreparedStatement.executeUpdate(Unknown Source)
	at com.mchange.v2.c3p0.impl.NewProxyPreparedStatement.executeUpdate(NewProxyPreparedStatement.java:105)
	at org.hibernate.id.insert.AbstractSelectingDelegate.performInsert(AbstractSelectingDelegate.java:57)
	... 54 more`

	msgs = writeByLine(m, data, time.Unix(0, 0))
	require.Len(t, msgs, 1)
	assert.Equal(t, data, msgs[0].Content)
}

func TestMultilineCollectorJS(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	m := NewMultilineCollector(ctx, 10*time.Millisecond, multilineCollectorLimit)
	defer cancel()

	data := `UnauthorizedException [Error]: jwt expired
    at AuthMixingGuard.canActivate (/app/dist/core/auth.guard.js:23:27)
    at GuardsConsumer.tryActivate (/app/node_modules/@nestjs/core/guards/guards-consumer.js:15:34)
    at canActivateFn (/app/node_modules/@nestjs/core/router/router-execution-context.js:134:59)
    at /app/node_modules/@nestjs/core/router/router-execution-context.js:42:37
    at AsyncLocalStorage.run (async_hooks.js:314:14)
    at /app/node_modules/@nestjs/core/router/router-proxy.js:9:23 {
  response: { statusCode: 401, message: 'jwt expired', error: 'Unauthorized' },`
	msgs := writeByLine(m, data, time.Unix(0, 0))
	require.Len(t, msgs, 1)
	assert.Equal(t, data, msgs[0].Content)

	data = `Error: Invalid IV length
    at Decipheriv.createCipherBase (internal/crypto/cipher.js:103:19)
    at Decipheriv.createCipherWithIV (internal/crypto/cipher.js:121:20)
    at new Decipheriv (internal/crypto/cipher.js:264:22)
    at Object.createDecipheriv (crypto.js:130:10)
    at AuthMixingGuard.canActivate (/app/dist/core/auth.guard.js:19:92)
    at GuardsConsumer.tryActivate (/app/node_modules/@nestjs/core/guards/guards-consumer.js:15:34)
    at canActivateFn (/app/node_modules/@nestjs/core/router/router-execution-context.js:134:59)
    at /app/node_modules/@nestjs/core/router/router-execution-context.js:42:37
::ffff:10.50.10.96 - - [15/Feb/2024:09:11:15 +0000] "POST /api/auth HTTP/1.1" 500 47 "https://example.com/" "Mozilla/5.0 ..."`
	msgs = writeByLine(m, data, time.Unix(0, 0))
	require.Len(t, msgs, 2)
	assert.Equal(t, data, msgs[0].Content+"\n"+msgs[1].Content)
}

func TestMultilineCollectorGO(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	m := NewMultilineCollector(ctx, 10*time.Millisecond, multilineCollectorLimit)
	defer cancel()

	// TODO: `panic` without timestamp in the first line

	data := `2024/02/16 15:01:22 http: panic serving 127.0.0.1:56889: runtime error: invalid memory address or nil pointer dereference
goroutine 675 [running]:
net/http.(*conn).serve.func1()
        /Users/user/sdk/go1.21.3/src/net/http/server.go:1868 +0xb0
panic({0x103383820?, 0x103ba2fa0?})
        /Users/user/sdk/go1.21.3/src/runtime/panic.go:920 +0x26c
github.com/coroot/coroot/api.(*Api).App(0x1034cd180?, {0x1034cab30, 0x1400239a0e0}, 0x1032707e0?)
        /Users/user/coroot/coroot/api/api.go:377 +0x228
net/http.HandlerFunc.ServeHTTP(0x140030e0800?, {0x1034cab30?, 0x1400239a0e0?}, 0x0?)
        /Users/user/sdk/go1.21.3/src/net/http/server.go:2136 +0x38
github.com/gorilla/mux.(*Router).ServeHTTP(0x140000fa180, {0x1034cab30, 0x1400239a0e0}, 0x140030e0600)
        /Users/user/go/pkg/mod/github.com/gorilla/mux@v1.8.0/mux.go:210 +0x194
net/http.serverHandler.ServeHTTP({0x1400474cb40?}, {0x1034cab30?, 0x1400239a0e0?}, 0x6?)
        /Users/user/sdk/go1.21.3/src/net/http/server.go:2938 +0xbc
net/http.(*conn).serve(0x14001dbe090, {0x1034cd180, 0x14000c412c0})
        /Users/user/sdk/go1.21.3/src/net/http/server.go:2009 +0x518
created by net/http.(*Server).Serve in goroutine 1
        /Users/user/sdk/go1.21.3/src/net/http/server.go:3086 +0x4cc`
	msgs := writeByLine(m, data, time.Unix(0, 0))
	require.Len(t, msgs, 1)
	assert.Equal(t, data, msgs[0].Content)
}
func TestMultilineCollectorLimit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	m := NewMultilineCollector(ctx, 10*time.Millisecond, 100)
	defer cancel()
	data := "I0215 12:33:07.230967 foo\n" + strings.Repeat("foo\n\n\n", 20)
	assert.Equal(t, 146, len(data))
	msgs := writeByLine(m, data, time.Unix(0, 0))
	require.Len(t, msgs, 1)
	assert.Equal(t, 100, len(msgs[0].Content))

	data = "I0215 12:33:07.230967" + strings.Repeat(" foo", 25)
	assert.Equal(t, 121, len(data))
	msgs = writeByLine(m, data, time.Unix(0, 0))
	require.Len(t, msgs, 1)
	assert.Equal(t, 100, len(msgs[0].Content))
}
