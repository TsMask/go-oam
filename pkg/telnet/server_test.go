package telnet

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func dial(t *testing.T, addr string) net.Conn {
	t.Helper()
	c, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	return c
}

// startServer 启动 s 监听任意端口。s.Handler 必须已设置。
func startServer(t *testing.T, s *Server) string {
	t.Helper()
	return startServerAddr(t, s, ":0")
}

func startServerAddr(t *testing.T, s *Server, address string) string {
	t.Helper()
	errCh := make(chan error, 1)
	go func() { errCh <- s.Listen(address) }()
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

func TestListen_NoHandler(t *testing.T) {
	s := &Server{}
	if err := s.Listen(":0"); !errors.Is(err, ErrNoHandler) {
		t.Fatalf("want ErrNoHandler, got %v", err)
	}
}

func TestServer_DefaultStateIsInit(t *testing.T) {
	s := &Server{}
	if s.State() != serverStateInit {
		t.Fatalf("want init, got %d", s.State())
	}
}

// === 基础 echo ===

func TestServer_BasicEcho(t *testing.T) {
	s := &Server{
		Handler: func(c *Conn) error {
			buf := make([]byte, 64)
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
	addr := startServer(t, s)
	defer s.Close()

	c := dial(t, addr)
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
		t.Fatalf("got %q", got.String())
	}
}

// === handler 返回 error → OnError ===

func TestServer_HandlerReturnError_ToOnError(t *testing.T) {
	var got atomic.Value
	myErr := errors.New("biz error")
	s := &Server{
		Handler: func(c *Conn) error { return myErr },
		OnError: func(err error) { got.Store(err) },
	}
	addr := startServer(t, s)
	defer s.Close()

	c := dial(t, addr)
	defer c.Close()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && got.Load() == nil {
		time.Sleep(10 * time.Millisecond)
	}
	if got.Load() == nil {
		t.Fatal("OnError not invoked")
	}
	if !errors.Is(got.Load().(error), myErr) {
		t.Fatalf("want myErr, got %v", got.Load())
	}
}

// === handler panic → OnError（带 stack） ===

func TestServer_HandlerPanic_ToOnError(t *testing.T) {
	var got atomic.Value
	s := &Server{
		Handler: func(c *Conn) error { panic("boom") },
		OnError: func(err error) { got.Store(err) },
	}
	addr := startServer(t, s)
	defer s.Close()

	c := dial(t, addr)
	defer c.Close()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && got.Load() == nil {
		time.Sleep(10 * time.Millisecond)
	}
	if got.Load() == nil {
		t.Fatal("OnError not invoked for panic")
	}
	errMsg := got.Load().(error).Error()
	if !strings.Contains(errMsg, "boom") || !strings.Contains(errMsg, "handler panic") {
		t.Fatalf("want panic msg with stack, got %q", errMsg)
	}
	if !strings.Contains(errMsg, "goroutine") {
		t.Fatalf("want stack trace, got %q", errMsg)
	}
}

// === OnError 未设置时 panic 不影响 server ===

func TestServer_PanicNoOnError_StillServes(t *testing.T) {
	s := &Server{
		Handler: func(c *Conn) error { panic("boom") },
	}
	addr := startServer(t, s)
	defer s.Close()

	c1 := dial(t, addr)
	_ = c1.Close()
	time.Sleep(100 * time.Millisecond)

	if s.State() != serverStateServing {
		t.Fatalf("want serving, got %d", s.State())
	}
	c2 := dial(t, addr)
	_ = c2.Close()
}

// === 生命周期 ===

func TestServer_CloseWaitsForHandler(t *testing.T) {
	handlerExited := make(chan struct{})
	s := &Server{
		Handler: func(c *Conn) error {
			<-c.Context().Done()
			close(handlerExited)
			return nil
		},
	}
	addr := startServer(t, s)

	c := dial(t, addr)
	defer c.Close()

	done := make(chan struct{})
	go func() {
		s.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Close did not return")
	}
	select {
	case <-handlerExited:
	default:
		t.Fatal("handler did not exit before Close returned")
	}
	if s.State() != serverStateClosed {
		t.Fatalf("want closed, got %d", s.State())
	}
}

func TestServer_CloseIdempotent(t *testing.T) {
	s := &Server{
		Handler: func(*Conn) error { return nil },
	}
	startServer(t, s)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Close()
		}()
	}
	wg.Wait()
	if s.State() != serverStateClosed {
		t.Fatalf("want closed, got %d", s.State())
	}
}

func TestServer_CloseBeforeListen(t *testing.T) {
	s := &Server{
		Handler: func(*Conn) error { return nil },
	}
	done := make(chan struct{})
	go func() {
		s.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close on unstarted server blocked")
	}
	if err := s.Listen(":0"); !errors.Is(err, ErrServerClosed) {
		t.Fatalf("want ErrServerClosed after Close, got %v", err)
	}
}

func TestServer_ListenTwice(t *testing.T) {
	s := &Server{
		Handler: func(*Conn) error { return nil },
	}
	startServer(t, s)
	defer s.Close()

	if err := s.Listen(":0"); !errors.Is(err, ErrAlreadyServing) {
		t.Fatalf("want ErrAlreadyServing, got %v", err)
	}
}

func TestServer_ListenAfterClose(t *testing.T) {
	s := &Server{
		Handler: func(*Conn) error { return nil },
	}
	startServer(t, s)
	s.Close()

	if err := s.Listen(":0"); !errors.Is(err, ErrServerClosed) {
		t.Fatalf("want ErrServerClosed, got %v", err)
	}
}

// === ctx 联动 ===

func TestServer_CtxCancelsOnClose(t *testing.T) {
	handlerCtxCanceled := make(chan struct{})
	s := &Server{
		Handler: func(c *Conn) error {
			<-c.Context().Done()
			close(handlerCtxCanceled)
			return nil
		},
	}
	addr := startServer(t, s)
	c := dial(t, addr)
	defer c.Close()

	s.Close()
	select {
	case <-handlerCtxCanceled:
	case <-time.After(time.Second):
		t.Fatal("handler ctx did not cancel")
	}
}

// === MaxConns ===

func TestServer_MaxConnsReject(t *testing.T) {
	hold := make(chan struct{})
	s := &Server{
		Handler:  func(c *Conn) error { <-hold; return nil },
		MaxConns: 1,
	}
	addr := startServer(t, s)
	defer func() {
		close(hold)
		s.Close()
	}()

	c1 := dial(t, addr)
	defer c1.Close()
	c2, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c2.Close()
	c2.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 128)
	n, _ := c2.Read(buf)
	if !strings.Contains(string(buf[:n]), "connection limit reached") {
		t.Fatalf("want rejection msg, got %q", buf[:n])
	}
}

// === MaxConns 拒绝时 OnError 收到通知 ===

func TestServer_MaxConnsReject_NotifiesOnError(t *testing.T) {
	hold := make(chan struct{})
	var got atomic.Value
	s := &Server{
		Handler:  func(c *Conn) error { <-hold; return nil },
		OnError:  func(err error) { got.Store(err) },
		MaxConns: 1,
	}
	addr := startServer(t, s)
	defer func() {
		close(hold)
		s.Close()
	}()

	c1 := dial(t, addr)
	defer c1.Close()
	c2, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c2.Close()
	c2.SetReadDeadline(time.Now().Add(time.Second))
	_, _ = c2.Read(make([]byte, 128))

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && got.Load() == nil {
		time.Sleep(10 * time.Millisecond)
	}
	if got.Load() == nil {
		t.Fatal("OnError not invoked for rejection")
	}
	if !strings.Contains(got.Load().(error).Error(), "max conns") {
		t.Fatalf("want max conns msg, got %v", got.Load())
	}
}

// === 并发压测 ===

func TestServer_ConcurrentConnections(t *testing.T) {
	var active atomic.Int32
	s := &Server{
		Handler: func(c *Conn) error {
			active.Add(1)
			defer active.Add(-1)
			buf := make([]byte, 4)
			if _, err := c.Read(buf); err != nil {
				return err
			}
			_, err := c.Write(buf)
			return err
		},
		MaxConns: 50,
	}
	addr := startServer(t, s)
	defer s.Close()

	const N = 100
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c, err := net.DialTimeout("tcp", addr, time.Second)
			if err != nil {
				return
			}
			defer c.Close()
			c.SetDeadline(time.Now().Add(2 * time.Second))
			_, _ = c.Write([]byte(fmt.Sprintf("p%02d", i)))
			buf := make([]byte, 4)
			_, _ = c.Read(buf)
		}(i)
	}
	wg.Wait()
	if active.Load() != 0 {
		t.Fatalf("active conns not zero: %d", active.Load())
	}
}

// === IPv6 ===

func TestServer_IPv6(t *testing.T) {
	s := &Server{
		Handler: func(c *Conn) error {
			_, err := c.Write([]byte("hello-v6"))
			return err
		},
	}
	addr := startServerAddr(t, s, "[::1]:0")
	defer s.Close()

	c, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Skipf("IPv6 not available: %v", err)
	}
	defer c.Close()
	c.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 16)
	n, err := c.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf[:n]) != "hello-v6" {
		t.Fatalf("got %q", buf[:n])
	}
}

// === Conn API ===

func TestConn_CloseIdempotent(t *testing.T) {
	closed := make(chan struct{})
	s := &Server{
		Handler: func(c *Conn) error {
			_ = c.Close()
			_ = c.Close()
			_ = c.Close()
			close(closed)
			return nil
		},
	}
	addr := startServer(t, s)
	defer s.Close()

	cc := dial(t, addr)
	defer cc.Close()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("handler did not exit")
	}
}

func TestServer_CloseMeansHandlersExited(t *testing.T) {
	var stillActive atomic.Int32
	s := &Server{
		Handler: func(c *Conn) error {
			stillActive.Add(1)
			<-c.Context().Done()
			stillActive.Add(-1)
			return nil
		},
	}
	addr := startServer(t, s)

	var clients []net.Conn
	for i := 0; i < 5; i++ {
		c := dial(t, addr)
		clients = append(clients, c)
	}
	defer func() {
		for _, c := range clients {
			_ = c.Close()
		}
	}()

	s.Close()
	if got := stillActive.Load(); got != 0 {
		t.Fatalf("after Close, %d handlers still active", got)
	}
}

func TestServer_HandlerReceivesServer(t *testing.T) {
	s := &Server{
		Handler: func(c *Conn) error {
			if c.Server() == nil {
				t.Errorf("Server() is nil")
			}
			if c.RemoteAddr() == nil {
				t.Errorf("RemoteAddr is nil")
			}
			_, err := io.WriteString(c, "ACK")
			return err
		},
	}
	addr := startServer(t, s)
	defer s.Close()

	c := dial(t, addr)
	defer c.Close()
	c.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 8)
	n, err := c.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf[:n]) != "ACK" {
		t.Fatalf("got %q", buf[:n])
	}
}
