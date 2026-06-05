package socket

import (
	"fmt"
	"net"
	"testing"
	"time"
)

// === TCP echo round-trip ===

func BenchmarkClientTCP_RoundTrip(b *testing.B) {
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
	}
	addr := startBenchServerTCP(b, s)
	defer s.Close()

	c := &ClientTCP{Addr: hostOf(addr), Port: portOf(addr), DialTimeout: 2 * time.Second}
	if err := c.Connect(); err != nil {
		b.Fatal(err)
	}
	defer c.Close()

	msg := []byte("hello")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := c.Write(msg); err != nil {
			b.Fatal(err)
		}
		got, err := c.Read()
		if err != nil {
			b.Fatal(err)
		}
		if string(got) != string(msg) {
			b.Fatalf("want %q, got %q", msg, got)
		}
	}
}

func BenchmarkServerTCP_ConcurrentEcho(b *testing.B) {
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
	}
	addr := startBenchServerTCP(b, s)
	defer s.Close()

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		c := &ClientTCP{Addr: hostOf(addr), Port: portOf(addr), DialTimeout: 2 * time.Second}
		if err := c.Connect(); err != nil {
			b.Fatal(err)
		}
		defer c.Close()
		msg := []byte("hello")
		for pb.Next() {
			if _, err := c.Write(msg); err != nil {
				b.Fatal(err)
			}
			if _, err := c.Read(); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// === UDP datagram round-trip ===

func BenchmarkClientUDP_RoundTrip(b *testing.B) {
	s := &ServerUDP{
		Handler: func(pc *PacketConn, data []byte, addr *net.UDPAddr) error {
			_, err := pc.WriteToUDP(data, addr)
			return err
		},
	}
	addr := startBenchServerUDP(b, s)
	defer s.Close()

	c := &ClientUDP{Addr: hostOf(addr), Port: portOf(addr), DialTimeout: 2 * time.Second}
	if err := c.Connect(); err != nil {
		b.Fatal(err)
	}
	defer c.Close()

	msg := []byte("hello")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := c.Write(msg); err != nil {
			b.Fatal(err)
		}
		if _, err := c.Read(); err != nil {
			b.Fatal(err)
		}
	}
}

// === helper ===

func startBenchServerTCP(b *testing.B, s *ServerTCP) string {
	b.Helper()
	errCh := make(chan error, 1)
	go func() { errCh <- s.Listen(":0") }()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s.ListenAddr() != nil {
			break
		}
		select {
		case err := <-errCh:
			b.Fatalf("Listen exited early: %v", err)
		case <-time.After(20 * time.Millisecond):
		}
	}
	if s.ListenAddr() == nil {
		b.Fatal("server did not start")
	}
	return s.ListenAddr().String()
}

func startBenchServerUDP(b *testing.B, s *ServerUDP) string {
	b.Helper()
	errCh := make(chan error, 1)
	go func() { errCh <- s.Listen(":0") }()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s.ListenAddr() != nil {
			break
		}
		select {
		case err := <-errCh:
			b.Fatalf("Listen exited early: %v", err)
		case <-time.After(20 * time.Millisecond):
		}
	}
	if s.ListenAddr() == nil {
		b.Fatal("server did not start")
	}
	return s.ListenAddr().String()
}

// 防止编译器认为 fmt 未使用
var _ = fmt.Sprintf
