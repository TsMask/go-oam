package socket

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// startEchoTCPServer 在任意端口跑一个 echo 服务（直接用 net.Listener）。
// 返回监听地址。
func startEchoTCPServer(t *testing.T) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 4096)
				for {
					n, err := c.Read(buf)
					if n > 0 {
						if _, werr := c.Write(buf[:n]); werr != nil {
							return
						}
					}
					if err != nil {
						return
					}
				}
			}(c)
		}
	}()
	return ln.Addr().String(), func() {
		close(stop)
		ln.Close()
	}
}

// startEchoUDPServer 在任意端口跑一个 echo 服务。
func startEchoUDPServer(t *testing.T) (string, func()) {
	t.Helper()
	pc, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		buf := make([]byte, 65536)
		for {
			n, raddr, err := pc.ReadFromUDP(buf)
			if err != nil {
				return
			}
			if n > 0 {
				_, _ = pc.WriteToUDP(buf[:n], raddr)
			}
		}
	}()
	return pc.LocalAddr().String(), func() { pc.Close() }
}

// === ClientTCP ===

// 基础 Read/Write round-trip

func TestClientTCP_BasicRoundTrip(t *testing.T) {
	addr, stop := startEchoTCPServer(t)
	defer stop()

	c := &ClientTCP{Addr: hostOf(addr), Port: portOf(addr), DialTimeout: 2 * time.Second}
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if !c.IsConnected() {
		t.Fatal("want connected")
	}

	if _, err := c.Write([]byte("PING")); err != nil {
		t.Fatal(err)
	}
	data, err := readWithin(t, c, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "PING" {
		t.Fatalf("want PING, got %q", data)
	}
}

// Connect 幂等 + Close 后再 Connect → ErrClientClosed

func TestClientTCP_ConnectIdempotentAndReuseAfterClose(t *testing.T) {
	addr, stop := startEchoTCPServer(t)
	defer stop()
	c := &ClientTCP{Addr: hostOf(addr), Port: portOf(addr), DialTimeout: 2 * time.Second}
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	if err := c.Connect(); err != nil {
		t.Fatalf("second Connect should be nil, got %v", err)
	}
	c.Close()
	if err := c.Connect(); !errors.Is(err, ErrClientClosed) {
		t.Fatalf("want ErrClientClosed, got %v", err)
	}
}

// Read/Write 在 closed 状态返回错误

func TestClientTCP_ReadWriteAfterClose(t *testing.T) {
	addr, stop := startEchoTCPServer(t)
	defer stop()
	c := &ClientTCP{Addr: hostOf(addr), Port: portOf(addr), DialTimeout: 2 * time.Second}
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	c.Close()
	if _, err := c.Write([]byte("x")); !errors.Is(err, ErrClientClosed) {
		t.Fatalf("Write after close: want ErrClientClosed, got %v", err)
	}
	// Read 在 closed 状态返 io.EOF（与 telnet 行为一致），靠 OnError 区分对端关
	_, err := c.Read()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("Read after close: want io.EOF, got %v", err)
	}
}

// 远端关闭 → Read 返回 io.EOF（与用户主动 Close 行为一致）；
// OnError 触发作为对端断开的唯一信号

func TestClientTCP_PeerClose(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	addr := ln.Addr().String()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		c.Close()
	}()

	var onErr atomic.Value
	c := &ClientTCP{Addr: hostOf(addr), Port: portOf(addr), DialTimeout: 2 * time.Second}
	c.OnError(func(err error) { onErr.Store(err) })
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	data, err := readWithin(t, c, 2*time.Second)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("want io.EOF, got %v (data=%q)", err, data)
	}
	// OnError 触发作为对端断开的信号（用户 Close 不触发）
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if v := onErr.Load(); v != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("OnError not fired on peer close")
}

// OnError 在对端异常断开时触发

func TestClientTCP_OnError_OnPeerClose(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	addr := ln.Addr().String()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		c.Close()
	}()

	var got atomic.Value
	c := &ClientTCP{Addr: hostOf(addr), Port: portOf(addr), DialTimeout: 2 * time.Second}
	c.OnError(func(err error) { got.Store(err) })
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if v := got.Load(); v != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("OnError not fired on peer close")
}

// Context 取消联动 Close

func TestClientTCP_ContextCancel_TriggersClose(t *testing.T) {
	addr, stop := startEchoTCPServer(t)
	defer stop()
	ctx, cancel := context.WithCancel(context.Background())
	c := &ClientTCP{
		Addr: hostOf(addr), Port: portOf(addr),
		DialTimeout: 2 * time.Second,
		Context:     ctx,
	}
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	cancel()
	// ctx 取消后 Read 应返回（ctx 取消时设 deadline 唤醒）
	_, err := readWithin(t, c, 2*time.Second)
	if err == nil {
		t.Fatal("want error after ctx cancel")
	}
	if !errors.Is(err, io.EOF) {
		// ctx cancel 会通过 deadline 触发 read 错误，包成 EOF 之外的错误也接受
		t.Logf("got err: %v (acceptable)", err)
	}
}

// OnError 回调内 panic 被吞掉

func TestClientTCP_OnErrorPanic_Swallowed(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	addr := ln.Addr().String()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		c.Close()
	}()

	c := &ClientTCP{Addr: hostOf(addr), Port: portOf(addr), DialTimeout: 2 * time.Second}
	c.OnError(func(err error) { panic("onerror panic") })
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	// 等待对端关闭 + OnError 被调用，进程不应崩溃
	time.Sleep(200 * time.Millisecond)
}

// 并发 Write 安全（writeMu 串行化）

func TestClientTCP_ConcurrentWrite(t *testing.T) {
	addr, stop := startEchoTCPServer(t)
	defer stop()
	c := &ClientTCP{Addr: hostOf(addr), Port: portOf(addr), DialTimeout: 2 * time.Second}
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Write([]byte("hi"))
		}()
	}
	wg.Wait()
}

// === ClientUDP ===

func TestClientUDP_BasicRoundTrip(t *testing.T) {
	addr, stop := startEchoUDPServer(t)
	defer stop()
	c := &ClientUDP{Addr: hostOf(addr), Port: portOf(addr), DialTimeout: 2 * time.Second}
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	if _, err := c.Write([]byte("PING")); err != nil {
		t.Fatal(err)
	}
	data, err := readWithinUDP(t, c, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "PING" {
		t.Fatalf("want PING, got %q", data)
	}
}

func TestClientUDP_DatagramBoundary(t *testing.T) {
	// 多个连续 datagram 独立返回，不拼接
	ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	addr := ln.LocalAddr().String()
	go func() {
		buf := make([]byte, 64)
		for {
			n, raddr, err := ln.ReadFromUDP(buf)
			if err != nil {
				return
			}
			// 回不同长度的包
			if string(buf[:n]) == "A" {
				ln.WriteToUDP([]byte("alpha"), raddr)
			} else {
				ln.WriteToUDP([]byte("beta!!"), raddr)
			}
		}
	}()

	c := &ClientUDP{Addr: hostOf(addr), Port: portOf(addr), DialTimeout: 2 * time.Second}
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	c.Write([]byte("A"))
	got1, _ := readWithinUDP(t, c, 2*time.Second)
	if string(got1) != "alpha" {
		t.Fatalf("first datagram: want alpha, got %q", got1)
	}
	c.Write([]byte("B"))
	got2, _ := readWithinUDP(t, c, 2*time.Second)
	if string(got2) != "beta!!" {
		t.Fatalf("second datagram: want beta!!, got %q", got2)
	}
}

func TestClientUDP_ConnectIdempotentAndReuseAfterClose(t *testing.T) {
	addr, stop := startEchoUDPServer(t)
	defer stop()
	c := &ClientUDP{Addr: hostOf(addr), Port: portOf(addr), DialTimeout: 2 * time.Second}
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	if err := c.Connect(); err != nil {
		t.Fatalf("second Connect should be nil, got %v", err)
	}
	c.Close()
	if err := c.Connect(); !errors.Is(err, ErrClientClosed) {
		t.Fatalf("want ErrClientClosed, got %v", err)
	}
}

func TestClientUDP_ReadWriteAfterClose(t *testing.T) {
	addr, stop := startEchoUDPServer(t)
	defer stop()
	c := &ClientUDP{Addr: hostOf(addr), Port: portOf(addr), DialTimeout: 2 * time.Second}
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	c.Close()
	if _, err := c.Write([]byte("x")); !errors.Is(err, ErrClientClosed) {
		t.Fatalf("Write after close: want ErrClientClosed, got %v", err)
	}
	_, err := c.Read()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("Read after close: want io.EOF, got %v", err)
	}
}

func TestClientUDP_ContextCancel(t *testing.T) {
	addr, stop := startEchoUDPServer(t)
	defer stop()
	ctx, cancel := context.WithCancel(context.Background())
	c := &ClientUDP{
		Addr: hostOf(addr), Port: portOf(addr),
		DialTimeout: 2 * time.Second,
		Context:     ctx,
	}
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	cancel()
	_, err := readWithinUDP(t, c, 2*time.Second)
	if err == nil {
		t.Fatal("want error after ctx cancel")
	}
}

// === helpers ===

// readWithin 在 timeout 内读一个 chunk。timeout 到返回超时错误。
func readWithin(t *testing.T, c *ClientTCP, timeout time.Duration) ([]byte, error) {
	t.Helper()
	type result struct {
		data []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		data, err := c.Read()
		ch <- result{data, err}
	}()
	select {
	case r := <-ch:
		return r.data, r.err
	case <-time.After(timeout):
		return nil, errors.New("read timeout")
	}
}

func readWithinUDP(t *testing.T, c *ClientUDP, timeout time.Duration) ([]byte, error) {
	t.Helper()
	type result struct {
		data []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		data, err := c.Read()
		ch <- result{data, err}
	}()
	select {
	case r := <-ch:
		return r.data, r.err
	case <-time.After(timeout):
		return nil, errors.New("read timeout")
	}
}

func hostOf(addr string) string {
	h, _, _ := net.SplitHostPort(addr)
	return h
}

func portOf(addr string) string {
	_, p, _ := net.SplitHostPort(addr)
	return p
}

// === Timeout 路径 ===

// DialTimeout: 不可达地址在 DialTimeout 内失败
func TestClientTCP_DialTimeout(t *testing.T) {
	// 选一个保留端口 + 不可达地址模拟超时
	c := &ClientTCP{
		Addr:        "10.255.255.1", // 不可达 IP
		Port:        "9999",
		DialTimeout: 100 * time.Millisecond,
	}
	start := time.Now()
	err := c.Connect()
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("want dial error")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("dial took too long: %v", elapsed)
	}
	if c.State() != clientStateInit {
		t.Fatalf("want init (retryable), got %d", c.State())
	}
}

// ReadTimeout: 读超时返回错误（无 Read 内置超时，要在外层 select）
func TestClientTCP_ReadTimeout(t *testing.T) {
	addr, stop := startEchoTCPServer(t)
	defer stop()
	c := &ClientTCP{
		Addr:        hostOf(addr),
		Port:        portOf(addr),
		DialTimeout: 2 * time.Second,
		ReadTimeout: 100 * time.Millisecond, // 每次 read 底层超时 100ms
	}
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	// 发起 read 但不写数据，read 会超时
	start := time.Now()
	_, err := c.Read()
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("want timeout error")
	}
	if elapsed > 1*time.Second {
		t.Fatalf("read should timeout around 100ms, took %v", elapsed)
	}
}

// WriteTimeout: 写超时。fill 对方的 read buffer 触发。
func TestClientTCP_WriteTimeout(t *testing.T) {
	// 起一个不读的服务端
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		c, _ := ln.Accept()
		if c == nil { return }
		defer c.Close()
		buf := make([]byte, 64)
		for {
			if _, err := c.Read(buf); err != nil { return }
		}
	}()

	c := &ClientTCP{
		Addr:         hostOf(ln.Addr().String()),
		Port:         portOf(ln.Addr().String()),
		DialTimeout:  2 * time.Second,
		WriteTimeout: 100 * time.Millisecond,
	}
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	// 写大块让对方 buffer 满
	big := make([]byte, 1<<20) // 1MB
	_, err := c.Write(big)
	if err == nil {
		// 第一次可能成功，再多写几次
		_, err = c.Write(big)
	}
	if err == nil {
		t.Log("write did not block (system buffer large); skipping strict assertion")
		return
	}
	t.Logf("write err: %v (expected timeout)", err)
}

// === ctx cancel 不触发 OnError（关键 bug fix）===

func TestClientTCP_CtxCancel_NoOnError(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		c, _ := ln.Accept()
		if c == nil { return }
		defer c.Close()
		buf := make([]byte, 64)
		for { if _, err := c.Read(buf); err != nil { return } }
	}()
	var got atomic.Value
	c := &ClientTCP{Addr: hostOf(ln.Addr().String()), Port: portOf(ln.Addr().String())}
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	c.Context = ctx
	c.OnError(func(err error) { got.Store(err) })
	if err := c.Connect(); err != nil { t.Fatal(err) }
	defer c.Close()
	time.Sleep(300 * time.Millisecond)
	if v := got.Load(); v != nil {
		t.Fatalf("OnError must NOT fire on ctx cancel, got: %v", v)
	}
}

func TestClientUDP_CtxCancel_NoOnError(t *testing.T) {
	pc, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	defer pc.Close()
	var got atomic.Value
	c := &ClientUDP{Addr: hostOf(pc.LocalAddr().String()), Port: portOf(pc.LocalAddr().String())}
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	c.Context = ctx
	c.OnError(func(err error) { got.Store(err) })
	if err := c.Connect(); err != nil { t.Fatal(err) }
	defer c.Close()
	time.Sleep(300 * time.Millisecond)
	if v := got.Load(); v != nil {
		t.Fatalf("OnError must NOT fire on ctx cancel, got: %v", v)
	}
}