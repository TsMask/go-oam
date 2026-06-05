package telnet

import (
	"bytes"
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

// === iacProcessor 单元测试 ===

func TestIACProcessor_PassThrough(t *testing.T) {
	var p iacProcessor
	got := p.feed([]byte("hello world"))
	if string(got) != "hello world" {
		t.Fatalf("want %q, got %q", "hello world", got)
	}
}

func TestIACProcessor_Strip2ByteCmd(t *testing.T) {
	var p iacProcessor
	got := p.feed([]byte{0x68, 0x69, 0xFF, 0xF0, 0x21})
	if string(got) != "hi!" {
		t.Fatalf("want %q, got %q", "hi!", got)
	}
}

func TestIACProcessor_Strip3ByteCmd(t *testing.T) {
	var p iacProcessor
	in := []byte{0x61, 0xFF, 0xFB, 0x01, 0x62, 0xFF, 0xFE, 0x1F, 0x63}
	got := p.feed(in)
	if string(got) != "abc" {
		t.Fatalf("want %q, got %q", "abc", got)
	}
}

func TestIACProcessor_StripSubneg(t *testing.T) {
	var p iacProcessor
	in := []byte{0x78, 0xFF, 0xFA, 0x1F, 0x00, 0x80, 0x00, 0x40, 0xFF, 0xF0, 0x79}
	got := p.feed(in)
	if string(got) != "xy" {
		t.Fatalf("want %q, got %q", "xy", got)
	}
}

func TestIACProcessor_EscapedIAC(t *testing.T) {
	var p iacProcessor
	in := []byte{0x61, 0xFF, 0xFF, 0x62}
	got := p.feed(in)
	if !bytes.Equal(got, []byte{'a', 0xFF, 'b'}) {
		t.Fatalf("want %q, got %q", []byte{'a', 0xFF, 'b'}, got)
	}
}

func TestIACProcessor_ChunkBoundary_2Byte(t *testing.T) {
	var p iacProcessor
	out := append(p.feed([]byte{0x68, 0xFF}), p.feed([]byte{0xF0, 0x69})...)
	if string(out) != "hi" {
		t.Fatalf("want %q, got %q", "hi", out)
	}
}

func TestIACProcessor_ChunkBoundary_3Byte(t *testing.T) {
	var p iacProcessor
	out := append(p.feed([]byte{0x68, 0xFF, 0xFB}), p.feed([]byte{0x01, 0x69})...)
	if string(out) != "hi" {
		t.Fatalf("want %q, got %q", "hi", out)
	}
}

func TestIACProcessor_ChunkBoundary_Subneg(t *testing.T) {
	var p iacProcessor
	parts := [][]byte{
		{0x78, 0xFF, 0xFA, 0x1F, 0x00},
		{0x80, 0x00, 0x40, 0xFF},
		{0xF0, 0x79},
	}
	var out []byte
	for _, part := range parts {
		out = append(out, p.feed(part)...)
	}
	if string(out) != "xy" {
		t.Fatalf("want %q, got %q", "xy", out)
	}
}

func TestIACProcessor_Disabled(t *testing.T) {
	var p iacProcessor
	p.disable()
	in := []byte{0x68, 0xFF, 0xFB, 0x01, 0x69}
	got := p.feed(in)
	if !bytes.Equal(got, in) {
		t.Fatalf("disabled should pass through, got %q", got)
	}
}

func TestIACProcessor_FastPath(t *testing.T) {
	var p iacProcessor
	in := []byte("hello world without any iac")
	got := p.feed(in)
	if !bytes.Equal(got, in) {
		t.Fatalf("want %q, got %q", in, got)
	}
	if len(got) > 0 && len(in) > 0 && &got[0] != &in[0] {
		t.Fatal("fast path should return same underlying array")
	}
}

// === Test helpers ===

// terminatorServer 回显收到的命令并附加 "> " 终止符
type terminatorServer struct {
	ln   net.Listener
	done chan struct{}
	once sync.Once
}

func newTerminatorServer(t *testing.T) *terminatorServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	es := &terminatorServer{ln: ln, done: make(chan struct{})}
	go es.serve()
	return es
}

func (e *terminatorServer) addr() string { return e.ln.Addr().String() }
func (e *terminatorServer) Close()       { e.once.Do(func() { e.ln.Close(); close(e.done) }) }

func (e *terminatorServer) serve() {
	for {
		conn, err := e.ln.Accept()
		if err != nil {
			return
		}
		go func() {
			defer conn.Close()
			buf := make([]byte, 4096)
			for {
				n, err := conn.Read(buf)
				if err != nil {
					return
				}
				if _, err := conn.Write(buf[:n]); err != nil {
					return
				}
				if _, err := conn.Write([]byte("> ")); err != nil {
					return
				}
			}
		}()
	}
}

func doneAtSuffix(suffix string) func([]byte) bool {
	s := []byte(suffix)
	return func(b []byte) bool { return bytes.HasSuffix(b, s) }
}

func newClient(t *testing.T, addr string) *Client {
	t.Helper()
	host, port, _ := net.SplitHostPort(addr)
	c := &Client{Addr: host, Port: port}
	if err := c.Connect(); err != nil {
		t.Fatalf("connect: %v", err)
	}
	return c
}

// === 生命周期：Connect/Close/IsConnected ===

func TestConnect_Idempotent(t *testing.T) {
	ts := newTerminatorServer(t)
	defer ts.Close()
	host, port, _ := net.SplitHostPort(ts.addr())
	c := &Client{Addr: host, Port: port}
	if err := c.Connect(); err != nil {
		t.Fatalf("first connect: %v", err)
	}
	if err := c.Connect(); err != nil {
		t.Fatalf("second connect should be idempotent: %v", err)
	}
	if !c.IsConnected() {
		t.Fatal("expected IsConnected after Connect")
	}
	c.Close()
}

func TestConnect_AfterClose_ReturnsErrClosed(t *testing.T) {
	ts := newTerminatorServer(t)
	defer ts.Close()
	c := newClient(t, ts.addr())
	c.Close()
	err := c.Connect()
	if !errors.Is(err, ErrClientClosed) {
		t.Fatalf("expected ErrClientClosed, got %v", err)
	}
}

func TestClose_BeforeConnect_NoOp(t *testing.T) {
	c := &Client{Addr: "127.0.0.1", Port: "23"}
	c.Close() // 不应 panic
	c.Close() // 多次调用安全
}

func TestClose_AfterClose_NoOp(t *testing.T) {
	ts := newTerminatorServer(t)
	defer ts.Close()
	c := newClient(t, ts.addr())
	c.Close()
	c.Close() // 多次调用安全
}

func TestIsConnected_ReflectsState(t *testing.T) {
	ts := newTerminatorServer(t)
	defer ts.Close()
	host, port, _ := net.SplitHostPort(ts.addr())
	c := &Client{Addr: host, Port: port}

	if c.IsConnected() {
		t.Fatal("expected not connected before Connect")
	}
	if err := c.Connect(); err != nil {
		t.Fatalf("connect: %v", err)
	}
	if !c.IsConnected() {
		t.Fatal("expected connected after Connect")
	}
	c.Close()
	if c.IsConnected() {
		t.Fatal("expected not connected after Close")
	}
}

// === 并发安全：Connect/Close 竞态 ===

func TestConcurrent_ConnectOnlyOneSucceeds(t *testing.T) {
	ts := newTerminatorServer(t)
	defer ts.Close()
	host, port, _ := net.SplitHostPort(ts.addr())
	c := &Client{Addr: host, Port: port}

	const N = 20
	var wg sync.WaitGroup
	errs := make(chan error, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- c.Connect()
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Connect should not error: %v", err)
		}
	}
	if !c.IsConnected() {
		t.Fatal("expected connected")
	}
	c.Close()
}

func TestConcurrent_ConnectAndClose(t *testing.T) {
	ts := newTerminatorServer(t)
	defer ts.Close()
	host, port, _ := net.SplitHostPort(ts.addr())

	for i := 0; i < 50; i++ {
		c := &Client{Addr: host, Port: port}
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); c.Connect() }()
		go func() { defer wg.Done(); c.Close() }()
		wg.Wait()
		// 状态必须是 init/connected/closed 之一
		if c.IsConnected() {
			// 如果还连着，再 Close 一次确保清理
			c.Close()
		}
	}
}

// === Read/Write 错误处理 ===

func TestRead_NotConnected_ErrNotConnected(t *testing.T) {
	c := &Client{Addr: "127.0.0.1", Port: "23"}
	_, err := c.Read()
	if !errors.Is(err, ErrClientNotConnected) {
		t.Fatalf("expected ErrClientNotConnected, got %v", err)
	}
}

func TestRead_AfterClose_EOForClosed(t *testing.T) {
	ts := newTerminatorServer(t)
	defer ts.Close()
	c := newClient(t, ts.addr())
	c.Close()
	_, err := c.Read()
	if !errors.Is(err, ErrClientClosed) && !errors.Is(err, io.EOF) {
		t.Fatalf("expected ErrClientClosed or io.EOF, got %v", err)
	}
}

func TestWrite_NotConnected_ErrNotConnected(t *testing.T) {
	c := &Client{Addr: "127.0.0.1", Port: "23"}
	_, err := c.Write([]byte("hello"))
	if !errors.Is(err, ErrClientNotConnected) {
		t.Fatalf("expected ErrClientNotConnected, got %v", err)
	}
}

func TestWrite_AfterClose_ErrClosed(t *testing.T) {
	ts := newTerminatorServer(t)
	defer ts.Close()
	c := newClient(t, ts.addr())
	c.Close()
	_, err := c.Write([]byte("hello"))
	if !errors.Is(err, ErrClientClosed) {
		t.Fatalf("expected ErrClientClosed, got %v", err)
	}
}

func TestRead_EOFOnClose(t *testing.T) {
	ts := newTerminatorServer(t)
	defer ts.Close()
	c := newClient(t, ts.addr())

	done := make(chan error, 1)
	go func() {
		_, err := c.Read()
		done <- err
	}()
	time.Sleep(20 * time.Millisecond)
	c.Close()

	select {
	case err := <-done:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("expected io.EOF, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Read did not return after Close")
	}
}

// === Exec 行为 ===

func TestExec_StripsIAC(t *testing.T) {
	ts := newTerminatorServer(t)
	defer ts.Close()
	c := newClient(t, ts.addr())
	defer c.Close()

	out, err := c.Exec("hello\r\n", doneAtSuffix("> "))
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if !bytes.Contains([]byte(out), []byte("hello")) {
		t.Fatalf("expected echo in %q", out)
	}
}

func TestExec_MaxReadTruncation(t *testing.T) {
	ts := newTerminatorServer(t)
	defer ts.Close()
	c := newClient(t, ts.addr())
	defer c.Close()
	c.MaxRead = 16

	_, err := c.Exec("0123456789ABCDEF\r\n", nil)
	if !errors.Is(err, ErrClientTruncated) {
		t.Fatalf("expected ErrClientTruncated, got %v", err)
	}
}

func TestExec_IACStripped(t *testing.T) {
	ts := newTerminatorServer(t)
	defer ts.Close()
	c := newClient(t, ts.addr())
	defer c.Close()

	cmd := "A\xFF\xFB\x01B\r\n"
	out, err := c.Exec(cmd, doneAtSuffix("> "))
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	for i := 0; i+1 < len(out); i++ {
		if out[i] == 0xFF && (out[i+1] == 0xFB || out[i+1] == 0xFD) {
			t.Fatalf("IAC not stripped: %q", out)
		}
	}
}

func TestExec_KeepIAC(t *testing.T) {
	ts := newTerminatorServer(t)
	defer ts.Close()
	c := newClient(t, ts.addr())
	defer c.Close()
	c.KeepIAC = true

	cmd := "A\xFF\xFB\x01B\r\n"
	out, err := c.Exec(cmd, doneAtSuffix("> "))
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if !bytes.Contains([]byte(out), []byte{0xFF, 0xFB, 0x01}) {
		t.Fatalf("KeepIAC should preserve IAC bytes: %q", out)
	}
}

func TestExec_NotConnected(t *testing.T) {
	c := &Client{Addr: "127.0.0.1", Port: "23"}
	_, err := c.Exec("hi\r\n", nil)
	if !errors.Is(err, ErrClientNotConnected) {
		t.Fatalf("expected ErrClientNotConnected, got %v", err)
	}
}

func TestExec_WithSelectTimeout(t *testing.T) {
	ts := newTerminatorServer(t)
	defer ts.Close()
	c := newClient(t, ts.addr())

	type result struct {
		out string
		err error
	}
	resCh := make(chan result, 1)
	go func() {
		out, err := c.Exec("hello\r\n", doneAtSuffix("> "))
		resCh <- result{out, err}
	}()

	select {
	case r := <-resCh:
		if r.err != nil {
			t.Fatalf("exec: %v", r.err)
		}
		if !bytes.Contains([]byte(r.out), []byte("hello")) {
			t.Fatalf("expected echo, got %q", r.out)
		}
	case <-time.After(2 * time.Second):
		c.Close()
		t.Fatal("exec timeout via select")
	}
}

// === 并发场景 ===

func TestConcurrent_SendMultiple(t *testing.T) {
	ts := newTerminatorServer(t)
	defer ts.Close()
	c := newClient(t, ts.addr())
	defer c.Close()

	const N = 20
	for i := 0; i < N; i++ {
		cmd := fmt.Sprintf("cmd-%d\r\n", i)
		out, err := c.Exec(cmd, doneAtSuffix("> "))
		if err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
		if !bytes.Contains([]byte(out), []byte(strings.TrimSpace(cmd))) {
			t.Fatalf("send %d: expected echo of %q, got %q", i, cmd, out)
		}
	}
}

func TestConcurrent_CloseInterruptsRead(t *testing.T) {
	ts := newTerminatorServer(t)
	defer ts.Close()
	c := newClient(t, ts.addr())

	done := make(chan error, 1)
	go func() {
		_, err := c.Read()
		done <- err
	}()
	time.Sleep(20 * time.Millisecond)
	c.Close()

	select {
	case err := <-done:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("expected io.EOF, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Read did not return after Close")
	}
}

func TestConcurrent_MultipleReaders(t *testing.T) {
	ts := newTerminatorServer(t)
	defer ts.Close()
	c := newClient(t, ts.addr())
	defer c.Close()

	const N = 5
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			done := make(chan error, 1)
			go func() {
				_, err := c.Read()
				done <- err
			}()
			select {
			case err := <-done:
				if err != nil && !errors.Is(err, io.EOF) {
					t.Errorf("read: %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Error("read timeout")
			}
		}()
	}

	for i := 0; i < N; i++ {
		if _, err := c.Write([]byte(fmt.Sprintf("cmd-%d\r\n", i))); err != nil {
			t.Fatalf("write: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	wg.Wait()
}

// === 基准测试 ===

func BenchmarkIACProcessor_FastPath(b *testing.B) {
	chunk := bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog "), 80)
	if bytes.IndexByte(chunk, iacByte) >= 0 {
		b.Fatal("test bug: chunk unexpectedly contains 0xFF")
	}
	var p iacProcessor
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.feed(chunk)
	}
}

func BenchmarkIACProcessor_WithIAC(b *testing.B) {
	chunk := bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog "), 80)
	chunk[100] = 0xFF
	chunk[101] = 0xFB
	chunk[102] = 0x01
	chunk[2000] = 0xFF
	chunk[2001] = 0xFA
	chunk[2002] = 0x1F
	chunk[2003] = 0x00
	chunk[2004] = 0x80
	chunk[2005] = 0xFF
	chunk[2006] = 0xF0
	var p iacProcessor
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.feed(chunk)
	}
}

// === Auth 错误路径 ===

func TestAuth_NotConnected(t *testing.T) {
	c := &Client{Addr: "127.0.0.1", Port: "23"}
	err := c.Auth("user", "pass")
	if !errors.Is(err, ErrClientNotConnected) {
		t.Fatalf("expected ErrClientNotConnected, got %v", err)
	}
}

func TestAuth_AfterClose_ErrClosed(t *testing.T) {
	ts := newTerminatorServer(t)
	defer ts.Close()
	c := newClient(t, ts.addr())
	c.Close()
	err := c.Auth("user", "pass")
	if !errors.Is(err, ErrClientClosed) {
		t.Fatalf("expected ErrClientClosed, got %v", err)
	}
}

func TestAuth_EmptyCreds_NoOp(t *testing.T) {
	ts := newTerminatorServer(t)
	defer ts.Close()
	host, port, _ := net.SplitHostPort(ts.addr())
	c := &Client{Addr: host, Port: port, AuthPromptWait: 50 * time.Millisecond}
	if err := c.Connect(); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer c.Close()
	if err := c.Auth("", ""); err != nil {
		t.Fatalf("empty creds should be no-op, got %v", err)
	}
	if err := c.Auth("user", ""); err != nil {
		t.Fatalf("empty password should be no-op, got %v", err)
	}
	if err := c.Auth("", "pass"); err != nil {
		t.Fatalf("empty user should be no-op, got %v", err)
	}
}

// TestIACProcessor_PendingOverflow 验证畸形/恶意 IAC 序列不会让 pending 无限增长
// 服务端持续发送 IAC 0xFF 而不跟随任何命令字节，pending 应当被 flush
func TestIACProcessor_PendingOverflow(t *testing.T) {
	var p iacProcessor
	// 5KB 的孤立 0xFF 字节（每个都是 IAC 起始），跨越 maxIACPending 阈值
	chunk := bytes.Repeat([]byte{iacByte}, 5*1024)
	got := p.feed(chunk)
	// 所有 0xFF 应当被作为孤立字节 flush 出来（因为 pending 超过阈值）
	if len(got) == 0 {
		t.Fatal("pending overflow should flush accumulated bytes, got empty output")
	}
	if len(p.pending) >= maxIACPending {
		t.Fatalf("pending not cleared after overflow: %d bytes", len(p.pending))
	}
}

// === P1: OnError 测试 ===

// kickServer 接受连接后立即关闭，模拟"远端主动断开"。
type kickServer struct {
	ln   net.Listener
	once sync.Once
}

func newKickServer(t *testing.T) *kickServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := &kickServer{ln: ln}
	go s.serve()
	return s
}

func (s *kickServer) addr() string { return s.ln.Addr().String() }

func (s *kickServer) Close() { s.once.Do(func() { _ = s.ln.Close() }) }

func (s *kickServer) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		_ = conn.Close() // 立即关闭，让 client 端 readLoop 收到 EOF
	}
}

// 远端断开触发 OnError
func TestClient_OnError_RemoteDisconnect(t *testing.T) {
	ts := newKickServer(t)
	defer ts.Close()

	c := &Client{}
	var got atomic.Value
	c.OnError(func(err error) { got.Store(err) })

	host, port, _ := net.SplitHostPort(ts.addr())
	c.Addr, c.Port = host, port
	if err := c.Connect(); err != nil {
		t.Fatalf("connect: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && got.Load() == nil {
		time.Sleep(20 * time.Millisecond)
	}
	if got.Load() == nil {
		t.Fatal("OnError not invoked on remote disconnect")
	}
	errMsg := got.Load().(error).Error()
	if !strings.Contains(errMsg, "read") {
		t.Fatalf("want 'read' in error, got %q", errMsg)
	}
	c.Close()
}

// 用户主动 Close 不触发 OnError
func TestClient_OnError_NotInvokedOnUserClose(t *testing.T) {
	ts := newTerminatorServer(t)
	defer ts.Close()
	c := newClient(t, ts.addr())

	var got atomic.Value
	c.OnError(func(err error) { got.Store(err) })

	c.Close()
	time.Sleep(200 * time.Millisecond)
	if got.Load() != nil {
		t.Fatalf("OnError should not fire on user Close, got %v", got.Load())
	}
}

// Connect 之后注册 OnError 也生效
func TestClient_OnError_RegisterAfterConnect(t *testing.T) {
	ts := newKickServer(t)
	defer ts.Close()

	c := &Client{}
	host, port, _ := net.SplitHostPort(ts.addr())
	c.Addr, c.Port = host, port
	if err := c.Connect(); err != nil {
		t.Fatalf("connect: %v", err)
	}

	var got atomic.Value
	c.OnError(func(err error) { got.Store(err) })

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && got.Load() == nil {
		time.Sleep(20 * time.Millisecond)
	}
	if got.Load() == nil {
		t.Fatal("OnError registered after Connect did not fire on remote disconnect")
	}
	c.Close()
}

// 远端断开后 state 自动切到 closed，IsConnected 返回 false
// Connect 在已 closed 时返回 ErrClientClosed（验证 P0 修复）
func TestClient_RemoteDisconnect_StateAutoClosed(t *testing.T) {
	ts := newKickServer(t)
	defer ts.Close()

	c := &Client{}
	host, port, _ := net.SplitHostPort(ts.addr())
	c.Addr, c.Port = host, port
	if err := c.Connect(); err != nil {
		t.Fatalf("connect: %v", err)
	}

	// 等 readLoop 收到 EOF 并触发 shutdown
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && c.State() != clientStateClosed {
		time.Sleep(20 * time.Millisecond)
	}
	if c.State() != clientStateClosed {
		t.Fatalf("want state=closed after remote disconnect, got %d", c.State())
	}
	if c.IsConnected() {
		t.Fatal("IsConnected should return false after remote disconnect")
	}

	// 已 closed 后调 Connect 应返回 ErrClientClosed（不再是幂等 nil）
	if err := c.Connect(); !errors.Is(err, ErrClientClosed) {
		t.Fatalf("want ErrClientClosed after remote disconnect, got %v", err)
	}

	// Close 幂等
	c.Close()
}
