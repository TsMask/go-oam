package socket

import (
	"errors"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func startUDPServer(t *testing.T, s *ServerUDP) string {
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

func TestServerUDP_ListenNoHandler(t *testing.T) {
	s := &ServerUDP{}
	if err := s.Listen(":0"); !errors.Is(err, ErrNoHandler) {
		t.Fatalf("want ErrNoHandler, got %v", err)
	}
}

func TestServerUDP_DefaultStateIsInit(t *testing.T) {
	s := &ServerUDP{}
	if s.State() != serverStateInit {
		t.Fatalf("want init, got %d", s.State())
	}
}

// === 基础 echo ===

func TestServerUDP_BasicEcho(t *testing.T) {
	s := &ServerUDP{
		Handler: func(pc *PacketConn, data []byte, addr *net.UDPAddr) error {
			_, err := pc.WriteToUDP(data, addr)
			return err
		},
	}
	addr := startUDPServer(t, s)
	defer s.Close()

	c, err := net.Dial("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := c.Write([]byte("PING")); err != nil {
		t.Fatal(err)
	}
	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 64)
	n, err := c.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf[:n]) != "PING" {
		t.Fatalf("want PING, got %q", buf[:n])
	}
}

// === handler 返回 error → OnError ===

func TestServerUDP_HandlerReturnError_ToOnError(t *testing.T) {
	var got atomic.Value
	myErr := errors.New("biz error")
	s := &ServerUDP{
		Handler: func(pc *PacketConn, data []byte, addr *net.UDPAddr) error { return myErr },
		OnError: func(err error) { got.Store(err) },
	}
	addr := startUDPServer(t, s)
	defer s.Close()

	c, err := net.Dial("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := c.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}

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

func TestServerUDP_HandlerPanic_ToOnError(t *testing.T) {
	var got atomic.Value
	s := &ServerUDP{
		Handler: func(pc *PacketConn, data []byte, addr *net.UDPAddr) error { panic("boom") },
		OnError: func(err error) { got.Store(err) },
	}
	addr := startUDPServer(t, s)
	defer s.Close()

	c, err := net.Dial("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Write([]byte("x"))

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

// === MaxConns 限流（UDP: 限 handler 并发，丢包时通过 OnError 通知）===

func TestServerUDP_MaxConns_Drops(t *testing.T) {
	holdCh := make(chan struct{})
	entered := make(chan struct{}, 1)
	var dropped atomic.Int32
	s := &ServerUDP{
		MaxConns: 1,
		Handler: func(pc *PacketConn, data []byte, addr *net.UDPAddr) error {
			select {
			case entered <- struct{}{}:
			default:
			}
			<-holdCh
			return nil
		},
		OnError: func(err error) {
			if strings.Contains(err.Error(), "max handlers") {
				dropped.Add(1)
			}
		},
	}
	addr := startUDPServer(t, s)
	defer s.Close()
	defer close(holdCh)

	c, err := net.Dial("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	// 第一个包占满 handler
	if _, err := c.Write([]byte("1")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("first handler did not enter")
	}

	// 第二个包应被丢弃
	if _, err := c.Write([]byte("2")); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if dropped.Load() > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("OnError not fired for dropped packet")
}

// === Close 幂等 ===

func TestServerUDP_Close_Idempotent(t *testing.T) {
	s := &ServerUDP{
		Handler: func(pc *PacketConn, data []byte, addr *net.UDPAddr) error { return nil },
	}
	startUDPServer(t, s)
	s.Close()
	s.Close() // 幂等
	if s.State() != serverStateClosed {
		t.Fatalf("want closed, got %d", s.State())
	}
	if err := s.Listen(":0"); !errors.Is(err, ErrServerClosed) {
		t.Fatalf("want ErrServerClosed, got %v", err)
	}
}

// === PacketConn 字段 ===

func TestServerUDP_PacketConnFields(t *testing.T) {
	var gotServer atomic.Value
	var gotLocal atomic.Value
	var gotRemote atomic.Value
	done := make(chan struct{})
	s := &ServerUDP{
		Handler: func(pc *PacketConn, data []byte, addr *net.UDPAddr) error {
			gotServer.Store(pc.Server())
			gotLocal.Store(pc.LocalAddr().String())
			gotRemote.Store(addr.String())
			pc.Close()
			close(done)
			return nil
		},
	}
	addr := startUDPServer(t, s)
	defer s.Close()

	c, err := net.Dial("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := c.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler not invoked")
	}
	if gotServer.Load() != s {
		t.Fatalf("Server() mismatch: %v vs %v", gotServer.Load(), s)
	}
	if gotLocal.Load() == nil {
		t.Fatal("LocalAddr nil")
	}
	if gotRemote.Load() == nil {
		t.Fatal("remote addr nil")
	}
}

// === 并发安全：多个 client 并发发包 ===

func TestServerUDP_ConcurrentPackets(t *testing.T) {
	const N = 50
	var received atomic.Int32
	s := &ServerUDP{
		Handler: func(pc *PacketConn, data []byte, addr *net.UDPAddr) error {
			received.Add(1)
			return nil
		},
	}
	addr := startUDPServer(t, s)
	defer s.Close()

	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			c, err := net.Dial("udp", addr)
			if err != nil {
				return
			}
			defer c.Close()
			_, _ = c.Write([]byte("x"))
		}()
	}
	wg.Wait()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if int(received.Load()) >= N {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("got %d/%d packets", received.Load(), N)
}

// === ConnCount 在 MaxConns=0 时也准确 ===

func TestServerUDP_ConnCount_NoMax(t *testing.T) {
	holdCh := make(chan struct{})
	entered := make(chan struct{}, 10)
	s := &ServerUDP{
		MaxConns: 0, // 故意不限流
		Handler: func(pc *PacketConn, data []byte, addr *net.UDPAddr) error {
			entered <- struct{}{}
			<-holdCh
			return nil
		},
	}
	addr := startUDPServer(t, s)
	defer s.Close()
	defer close(holdCh)

	// 发 3 个包
	for i := 0; i < 3; i++ {
		c, _ := net.Dial("udp", addr)
		c.Write([]byte("x"))
		c.Close()
	}
	for i := 0; i < 3; i++ {
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			t.Fatalf("handler %d not entered", i)
		}
	}
	if got := s.ConnCount(); got != 3 {
		t.Fatalf("ConnCount want 3, got %d", got)
	}
}
// === Listen 失败不 leak ctx ===

func TestServerUDP_ListenFail_NoCtxLeak(t *testing.T) {
	// 占用端口让 Listen 失败
	pc, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	addr := pc.LocalAddr().String()

	s := &ServerUDP{
		Handler: func(pc *PacketConn, data []byte, addr *net.UDPAddr) error { return nil },
	}
	err = s.Listen(addr)
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