package socket

import (
	"errors"
	"fmt"
	"net"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// === helpers ===

// eventually 在 timeout 内反复 poll fn，fn 返 true 视为条件达成。
// 替代 time.Sleep(50ms); check; fail 这种 flake 友好性差的写法。
func eventually(t *testing.T, timeout time.Duration, interval time.Duration, fn func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return true
		}
		time.Sleep(interval)
	}
	return fn()
}

// goroutineDelta 跑 fn 前后对比 goroutine 数；返回净增量。
func goroutineDelta(t *testing.T, fn func()) int {
	t.Helper()
	runtime.GC()
	time.Sleep(20 * time.Millisecond)
	before := runtime.NumGoroutine()
	fn()
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	after := runtime.NumGoroutine()
	return after - before
}

// === ClientTCP goroutine leak ===

func TestStability_ClientTCP_NoGoroutineLeak(t *testing.T) {
	addr, stop := startEchoTCPServer(t)
	defer stop()

	delta := goroutineDelta(t, func() {
		c := &ClientTCP{Addr: hostOf(addr), Port: portOf(addr), DialTimeout: 2 * time.Second}
		if err := c.Connect(); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Write([]byte("ping")); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Read(); err != nil {
			t.Fatal(err)
		}
		c.Close()
	})
	if delta > 0 {
		t.Fatalf("goroutine leak: delta=%d", delta)
	}
}

func TestStability_ClientUDP_NoGoroutineLeak(t *testing.T) {
	addr, stop := startEchoUDPServer(t)
	defer stop()

	delta := goroutineDelta(t, func() {
		c := &ClientUDP{Addr: hostOf(addr), Port: portOf(addr), DialTimeout: 2 * time.Second}
		if err := c.Connect(); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Write([]byte("ping")); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Read(); err != nil {
			t.Fatal(err)
		}
		c.Close()
	})
	if delta > 0 {
		t.Fatalf("goroutine leak: delta=%d", delta)
	}
}

// === ServerTCP goroutine leak ===

func TestStability_ServerTCP_NoGoroutineLeak(t *testing.T) {
	s := &ServerTCP{
		Handler: func(c *Conn) error {
			buf := make([]byte, 64)
			c.Read(buf)
			return nil
		},
	}
	addr := startTCPServer(t, s)

	delta := goroutineDelta(t, func() {
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				conn, _ := net.DialTimeout("tcp", addr, 2*time.Second)
				if conn != nil {
					conn.Close()
				}
			}()
		}
		wg.Wait()
		eventually(t, time.Second, 10*time.Millisecond, func() bool {
			return s.ConnCount() == 0
		})
		s.Close()
	})
	if delta > 0 {
		t.Fatalf("goroutine leak: delta=%d", delta)
	}
	if s.State() != serverStateClosed {
		t.Fatalf("state want closed, got %d", s.State())
	}
}

func TestStability_ServerUDP_NoGoroutineLeak(t *testing.T) {
	s := &ServerUDP{
		Handler: func(pc *PacketConn, data []byte, addr *net.UDPAddr) error { return nil },
	}
	addr := startUDPServer(t, s)

	delta := goroutineDelta(t, func() {
		for i := 0; i < 10; i++ {
			c, _ := net.Dial("udp", addr)
			c.Write([]byte("x"))
			c.Close()
		}
		eventually(t, time.Second, 10*time.Millisecond, func() bool {
			return s.ConnCount() == 0
		})
		s.Close()
	})
	if delta > 0 {
		t.Fatalf("goroutine leak: delta=%d", delta)
	}
}

// === Stress：高并发 TCP echo ===

func TestStability_TCP_ConcurrentClients(t *testing.T) {
	s := &ServerTCP{
		Handler: func(c *Conn) error {
			buf := make([]byte, 1024)
			for {
				n, err := c.Read(buf)
				if err != nil {
					return err
				}
				if _, err := c.Write(buf[:n]); err != nil {
					return err
				}
			}
		},
		MaxConns: 100,
	}
	addr := startTCPServer(t, s)
	defer s.Close()

	const N = 50
	const M = 10
	var wg sync.WaitGroup
	var ok, fail atomic.Int64
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(id int) {
			defer wg.Done()
			c := &ClientTCP{Addr: hostOf(addr), Port: portOf(addr), DialTimeout: 5 * time.Second}
			if err := c.Connect(); err != nil {
				fail.Add(1)
				return
			}
			defer c.Close()
			for j := 0; j < M; j++ {
				msg := []byte(fmt.Sprintf("c%d-m%d", id, j))
				if _, err := c.Write(msg); err != nil {
					fail.Add(1)
					return
				}
				got, err := c.Read()
				if err != nil {
					fail.Add(1)
					return
				}
				if string(got) != string(msg) {
					fail.Add(1)
					return
				}
				ok.Add(1)
			}
		}(i)
	}
	wg.Wait()
	if int(fail.Load()) > 0 {
		t.Fatalf("stress: %d failures, %d ok", fail.Load(), ok.Load())
	}
	if got := ok.Load(); got != int64(N*M) {
		t.Fatalf("stress: want %d ok, got %d", N*M, got)
	}
}

// === Stress：高并发 UDP 包 ===

func TestStability_UDP_ConcurrentClients(t *testing.T) {
	s := &ServerUDP{
		Handler: func(pc *PacketConn, data []byte, addr *net.UDPAddr) error {
			_, err := pc.WriteToUDP(data, addr)
			return err
		},
	}
	addr := startUDPServer(t, s)
	defer s.Close()

	const N = 30
	const M = 5
	var wg sync.WaitGroup
	var ok, fail atomic.Int64
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(id int) {
			defer wg.Done()
			c := &ClientUDP{Addr: hostOf(addr), Port: portOf(addr), DialTimeout: 2 * time.Second}
			if err := c.Connect(); err != nil {
				fail.Add(1)
				return
			}
			defer c.Close()
			for j := 0; j < M; j++ {
				msg := []byte(fmt.Sprintf("c%d-m%d", id, j))
				if _, err := c.Write(msg); err != nil {
					fail.Add(1)
					return
				}
				got, err := readWithinUDP(t, c, 2*time.Second)
				if err != nil {
					fail.Add(1)
					return
				}
				if string(got) != string(msg) {
					fail.Add(1)
					return
				}
				ok.Add(1)
			}
		}(i)
	}
	wg.Wait()
	if got := ok.Load(); got < int64(N*M*8/10) {
		t.Fatalf("stress: want >= %d ok, got %d (fail=%d)", N*M*8/10, got, fail.Load())
	}
}

// === Stress：MaxConns 拒绝 + OnError 统计 ===

func TestStability_TCP_MaxConnsUnderLoad(t *testing.T) {
	block := make(chan struct{})
	entered := make(chan struct{}, 200)
	var rejected atomic.Int64
	s := &ServerTCP{
		MaxConns: 5,
		Handler: func(c *Conn) error {
			select {
			case entered <- struct{}{}:
			default:
			}
			<-block
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
	defer close(block)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, _ := net.DialTimeout("tcp", addr, 2*time.Second)
			if conn != nil {
				defer conn.Close()
				time.Sleep(500 * time.Millisecond)
			}
		}()
	}
	wg.Wait()
	if rejected.Load() == 0 {
		t.Fatal("expected some connections to be rejected")
	}
}

// === Read 在长时间无数据时不泄漏 ===

func TestStability_LongReadNoData(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		c, _ := ln.Accept()
		if c != nil {
			defer c.Close()
			time.Sleep(2 * time.Second)
		}
	}()

	c := &ClientTCP{Addr: hostOf(ln.Addr().String()), Port: portOf(ln.Addr().String())}
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	var gotEOF atomic.Int64
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.Read()
			if errors.Is(err, ErrClientClosed) || err == nil {
				return
			}
			gotEOF.Add(1)
		}()
	}
	time.Sleep(100 * time.Millisecond)
	c.Close()
	wg.Wait()
}
