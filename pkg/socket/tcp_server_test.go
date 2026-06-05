package socket

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// startTCPServer 启动 s 监听任意端口，阻塞直到 ListenAddr 可读或超时。
// s.Handler / s.OnError 等字段必须已设置。
func startTCPServer(t *testing.T, s *ServerTCP) string {
	t.Helper()
	errCh := make(chan error, 1)
	go func() { errCh <- s.Listen(":0") }()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s.ListenAddr() != nil {
			break
		}
		select {
		case err := <-errCh:
			t.Fatalf("Listen exited early: %v", err)
		case <-time.After(20 * time.Millisecond):
		}
	}
	if s.ListenAddr() == nil {
		t.Fatalf("server did not start")
	}
	return s.ListenAddr().String()
}

// === Listen 校验 ===

func TestServerTCP_ListenNoHandler(t *testing.T) {
	s := &ServerTCP{}
	if err := s.Listen(":0"); !errors.Is(err, ErrNoHandler) {
		t.Fatalf("want ErrNoHandler, got %v", err)
	}
}

func TestServerTCP_DefaultStateIsInit(t *testing.T) {
	s := &ServerTCP{}
	if s.State() != serverStateInit {
		t.Fatalf("want init, got %d", s.State())
	}
}

// === 基础 echo ===

func TestServerTCP_BasicEcho(t *testing.T) {
	s := &ServerTCP{
		Handler: func(c *Conn) error {
			buf := make([]byte, 4096)
			n, err := c.Read(buf)
			if err != nil {
				return err
			}
			if _, err := c.Write(buf[:n]); err != nil {
				return err
			}
			_, err = c.Write([]byte("DONE"))
			return err
		},
	}
	addr := startTCPServer(t, s)
	defer s.Close()

	c, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := c.Write([]byte("PING")); err != nil {
		t.Fatal(err)
	}
	c.SetReadDeadline(time.Now().Add(time.Second))
	var got strings.Builder
	buf := make([]byte, 64)
	for got.Len() < 8 {
		n, err := c.Read(buf)
		if n > 0 {
			got.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
	if got.String() != "PINGDONE" {
		t.Fatalf("want %q, got %q", "PINGDONE", got.String())
	}
}

// === handler 返回 error → OnError ===

func TestServerTCP_HandlerReturnError_ToOnError(t *testing.T) {
	var got atomic.Value
	myErr := errors.New("biz error")
	s := &ServerTCP{
		Handler: func(c *Conn) error { return myErr },
		OnError: func(err error) { got.Store(err) },
	}
	addr := startTCPServer(t, s)
	defer s.Close()

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()

	// 等 OnError 被触发
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if v := got.Load(); v != nil {
			if !errors.Is(v.(error), myErr) {
				t.Fatalf("want myErr, got %v", v)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("OnError not invoked")
}

// === handler panic → OnError（带 stack）===

func TestServerTCP_HandlerPanic_ToOnError(t *testing.T) {
	var got atomic.Value
	s := &ServerTCP{
		Handler: func(c *Conn) error { panic("boom") },
		OnError: func(err error) { got.Store(err) },
	}
	addr := startTCPServer(t, s)
	defer s.Close()

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if v := got.Load(); v != nil {
			msg := v.(error).Error()
			if !strings.Contains(msg, "handler panic") || !strings.Contains(msg, "boom") {
				t.Fatalf("want panic info, got %v", v)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("OnError not invoked")
}

// === OnError 内 panic 被吞掉 ===

func TestServerTCP_OnErrorPanic_Swallowed(t *testing.T) {
	s := &ServerTCP{
		Handler: func(c *Conn) error { return errors.New("err") },
		OnError: func(err error) { panic("onerror panic") },
	}
	addr := startTCPServer(t, s)
	defer s.Close()

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()

	// 等 handler 退出且 OnError 被吞掉（不会让后续连接崩）
	if !eventually(t, time.Second, 20*time.Millisecond, func() bool {
		// 等 handler 退出后 ConnCount 回到 0
		return s.ConnCount() == 0
	}) {
		t.Fatalf("handler did not exit")
	}
	if s.State() != serverStateServing {
		t.Fatalf("server crashed from OnError panic: state=%d", s.State())
	}
}

// === MaxConns 拒绝 ===

func TestServerTCP_MaxConns_Rejects(t *testing.T) {
	blockCh := make(chan struct{})
	released := make(chan struct{})
	release := false
	var rejected atomic.Int32

	s := &ServerTCP{
		MaxConns: 1,
		Handler: func(c *Conn) error {
			rejected.Add(1)
			<-blockCh
			if !release {
				close(released)
				release = true
			}
			return nil
		},
		OnError: func(err error) {
			if strings.Contains(err.Error(), "max conns") {
				rejected.Add(1)
			}
		},
	}
	addr := startTCPServer(t, s)
	defer s.Close()

	// 第一个连接占满
	conn1, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn1.Close()
	// 等 handler 进入
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s.ConnCount() == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if s.ConnCount() != 1 {
		t.Fatalf("want 1 conn, got %d", s.ConnCount())
	}

	// 第二个连接应被拒绝
	conn2, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close()
	// 拒绝路径会写一行 ERROR 再关，客户端应能读出再读到 EOF
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 256)
	var got strings.Builder
	for {
		n, err := conn2.Read(buf)
		if n > 0 {
			got.Write(buf[:n])
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				t.Fatalf("want EOF, got %v", err)
			}
			break
		}
	}
	if !strings.Contains(got.String(), "ERROR") {
		t.Fatalf("want reject message, got %q", got.String())
	}

	// 释放 handler，确认 OnError 至少触发一次"max conns"
	close(blockCh)
	<-released
	if !eventually(t, time.Second, 10*time.Millisecond, func() bool {
		return rejected.Load() > 0
	}) {
		t.Fatal("expected OnError to fire on max conns reject")
	}
}

// === Close 等待 handler 退出 ===

func TestServerTCP_Close_WaitsForHandler(t *testing.T) {
	handlerExited := make(chan struct{})
	s := &ServerTCP{
		Handler: func(c *Conn) error {
			defer close(handlerExited)
			// 模拟长任务：阻塞 200ms
			time.Sleep(200 * time.Millisecond)
			return nil
		},
	}
	addr := startTCPServer(t, s)

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()

	// handler 尚未退出时 Close 不应完成
	go func() {
		time.Sleep(50 * time.Millisecond)
		s.Close()
	}()
	select {
	case <-handlerExited:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not exit before server Close returned")
	}
}

// === Close 幂等 + Listen 之后 Close 再次 Listen → ErrServerClosed ===

func TestServerTCP_Close_IdempotentAndReListen(t *testing.T) {
	s := &ServerTCP{
		Handler: func(c *Conn) error { return nil },
	}
	startTCPServer(t, s)
	s.Close()
	s.Close() // 第二次幂等

	if err := s.Listen(":0"); !errors.Is(err, ErrServerClosed) {
		t.Fatalf("want ErrServerClosed, got %v", err)
	}
}

// === ListenAddr 端口 0 解析 ===

func TestServerTCP_ListenAddr_PortZero(t *testing.T) {
	s := &ServerTCP{
		Handler: func(c *Conn) error { return nil },
	}
	addr := startTCPServer(t, s)
	defer s.Close()
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	if host == "" || port == "0" {
		t.Fatalf("ListenAddr not bound: %s", addr)
	}
}

// === ConnCount 跟踪活跃连接 ===

func TestServerTCP_ConnCount(t *testing.T) {
	s := &ServerTCP{
		Handler: func(c *Conn) error {
			buf := make([]byte, 64)
			n, err := c.Read(buf)
			t.Logf("handler read n=%d err=%v", n, err)
			return nil
		},
	}
	addr := startTCPServer(t, s)
	defer s.Close()

	conn1, _ := net.DialTimeout("tcp", addr, 2*time.Second)
	conn2, _ := net.DialTimeout("tcp", addr, 2*time.Second)
	defer conn2.Close()

	for time.Now().Add(2*time.Second).After(time.Now()) {
		if s.ConnCount() == 2 { break }
		time.Sleep(10*time.Millisecond)
	}
	if s.ConnCount() != 2 {
		t.Fatalf("want 2 conns, got %d", s.ConnCount())
	}
	t.Logf("about to close conn1, count=%d", s.ConnCount())
	conn1.Close()
	for i := 0; i < 20; i++ {
		t.Logf("poll %d: count=%d", i, s.ConnCount())
		if s.ConnCount() == 1 { return }
		time.Sleep(100*time.Millisecond)
	}
	t.Fatalf("ConnCount did not decrease: %d", s.ConnCount())
}

// === Server 端 Context 取消联动 Close ===

func TestServerTCP_ContextCancel_TriggersClose(t *testing.T) {
	addrCh := make(chan string, 1)
	s := &ServerTCP{
		Context: func() context.Context {
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				time.Sleep(50 * time.Millisecond)
				cancel()
			}()
			return ctx
		}(),
		Handler: func(c *Conn) error {
			addrCh <- c.RemoteAddr().String()
			// ctx 取消后 read 应立即返回
			buf := make([]byte, 64)
			_, err := c.Read(buf)
			if err == nil {
				t.Errorf("want error from ctx cancel, got nil")
			}
			return nil
		},
	}

	// 直接 Listen（同 goroutine，Listen 内部会阻塞到 close）
	errCh := make(chan error, 1)
	go func() { errCh <- s.Listen(":0") }()
	deadline := time.Now().Add(2 * time.Second)
	var addr string
	for time.Now().Before(deadline) {
		if la := s.ListenAddr(); la != nil {
			addr = la.String()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if addr == "" {
		t.Fatal("server did not start")
	}

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	select {
	case <-addrCh:
	case <-time.After(2 * time.Second):
		t.Fatal("handler not invoked")
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Listen err: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Listen did not return after ctx cancel")
	}
	if s.State() != serverStateClosed {
		t.Fatalf("want closed, got %d", s.State())
	}
}

// === 并发安全：Server.OnError 来自 accept 与 handler 两路 ===

func TestServerTCP_ConcurrentAccess(t *testing.T) {
	var wg sync.WaitGroup
	var errors_ atomic.Int32
	s := &ServerTCP{
		Handler: func(c *Conn) error {
			defer wg.Done()
			buf := make([]byte, 64)
			_, err := c.Read(buf)
			if err == nil {
				return nil
			}
			return err
		},
		OnError: func(err error) {
			errors_.Add(1)
		},
		MaxConns: 50,
	}
	addr := startTCPServer(t, s)
	defer s.Close()

	const N = 30
	wg.Add(N)
	for i := 0; i < N; i++ {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		go func(c net.Conn) {
			defer c.Close()
			_, _ = c.Write([]byte("x"))
		}(conn)
	}
	wg.Wait()
}

// === Listen 失败不 leak ctx ===

func TestServerTCP_ListenFail_NoCtxLeak(t *testing.T) {
	// 占用端口让 Listen 失败
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	s := &ServerTCP{
		Handler: func(c *Conn) error { return nil },
	}
	err = s.Listen(ln.Addr().String())
	if err == nil {
		t.Fatal("want listen error")
	}
	if s.State() != serverStateClosed {
		t.Fatalf("state want closed, got %d", s.State())
	}
	s.listenerMu.RLock()
	ctx := s.serverCtx
	cancel := s.serverCancel
	s.listenerMu.RUnlock()
	if ctx != nil {
		t.Fatalf("serverCtx should be nil after Listen failure (avoid ctx leak), got %v", ctx)
	}
	if cancel != nil {
		t.Fatalf("serverCancel should be nil after Listen failure, got %v", cancel)
	}
}