package main

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/tsmask/go-oam"

	"github.com/tsmask/go-oam/framework/telnet"
)

var wg sync.WaitGroup

func main() {
	// 开启OAM
	wg.Add(1)
	go func() {
		defer wg.Done()

		o := oam.New(&oam.Opts{
			Dev: true,
			License: oam.License{
				NeType:     "NE",
				Version:    "1.0",
				SerialNum:  "1234567890",
				ExpiryDate: "2025-12-31",
				NbNumber:   10,
				UeNumber:   100,
			},
			ListenArr: []oam.Listen{
				{
					Addr:   "0.0.0.0:29565",
					Schema: "http",
				},
				{
					Addr:   "0.0.0.0:29567",
					Schema: "https",
					Cert:   "./dev/certs/www.oam.net.crt",
					Key:    "./dev/certs/www.oam.net.key",
				},
			},
		})
		if err := o.Run(); err != nil {
			fmt.Printf("oam run fail: %s\n", err.Error())
		}
	}()

	// 模拟telnet
	wg.Add(1)
	go func() {
		defer wg.Done()
		TelnetServer()
	}()

	// 模拟服务
	wg.Add(1)
	go func() {
		defer wg.Done()
		TestServer()
	}()

	wg.Wait()
}

// TestServer 模块
func TestServer() {
	fmt.Println("开始加载 ====> test 服务")

	// 创建周期为5秒的定时器
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop() // 确保在函数退出时停止定时器

	// 使用goroutine处理定时任务
	for t := range ticker.C {
		fmt.Println("test 服务", t.Format(time.RFC3339))
	}
}

// TelnetServer 模块
func TelnetServer() error {
	fmt.Println("开始加载 ====> telnet 服务")

	// 初始化服务
	telnetService := telnet.Server{
		Addr: "127.0.0.1",
		Port: "4100",
	}
	if err := telnetService.Listen(); err != nil {
		fmt.Printf("socket tcp init fail: %s\n", err.Error())
		return err
	}
	// 接收处理TCP数据
	telnetService.Resolve(func(conn net.Conn, err error) {
		if err != nil {
			fmt.Printf("TCP Resolve %s\n", err.Error())
			return
		}
		fmt.Println("[Telnet] TCP Accept from:", conn.RemoteAddr().String())
		conn.Write([]byte("hello world"))
	})
	return nil
}
