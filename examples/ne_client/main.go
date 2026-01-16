package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/framework/ws/protocol"
)

// Config 客户端配置
type Config struct {
	ServerURL         string
	NEID              string
	ReconnectInit     time.Duration
	ReconnectMax      time.Duration
	HeartbeatInterval time.Duration
}

// NEClient 网元客户端
type NEClient struct {
	config Config
	conn   *ws.ClientConn
	stopCh chan struct{}
}

// NewNEClient 创建实例
func NewNEClient(cfg Config) *NEClient {
	return &NEClient{
		config: cfg,
		stopCh: make(chan struct{}),
	}
}

// Start 启动客户端
func (c *NEClient) Start() {
	go c.runLoop()
}

// Stop 停止客户端
func (c *NEClient) Stop() {
	close(c.stopCh)
	if c.conn != nil {
		c.conn.Close()
	}
}

// runLoop 主循环：处理连接和重连
func (c *NEClient) runLoop() {
	retryCount := 0
	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		// 1. 尝试连接
		err := c.connect()
		if err == nil {
			// 连接成功，重置重试计数
			retryCount = 0
			// 阻塞等待连接断开
			<-c.conn.CloseSignal()
			log.Printf("[NE] Connection closed")
		} else {
			log.Printf("[NE] Connect failed: %v", err)
		}

		// 2. 计算退避时间
		retryCount++
		// 指数退避: Init * 2^(n-1)
		backoff := float64(c.config.ReconnectInit) * math.Pow(2, float64(retryCount-1))
		if backoff > float64(c.config.ReconnectMax) {
			backoff = float64(c.config.ReconnectMax)
		}
		sleepDuration := time.Duration(backoff)

		log.Printf("[NE] Reconnecting in %v (Attempt %d)...", sleepDuration, retryCount)

		// 3. 等待重连
		select {
		case <-time.After(sleepDuration):
		case <-c.stopCh:
			return
		}
	}
}

// connect 建立单次连接
func (c *NEClient) connect() error {
	log.Printf("[NE] Connecting to %s...", c.config.ServerURL)
	start := time.Now()

	// 每次连接创建新的ClientConn实例
	c.conn = &ws.ClientConn{
		Url:       c.config.ServerURL,
		Heartbeat: 30 * time.Second, // WS层心跳(Ping/Pong)，保持TCP活跃
	}

	if err := c.conn.Connect(); err != nil {
		return err
	}

	latency := time.Since(start)
	if latency > 3*time.Second {
		log.Printf("[NE] Warning: Connection slow (%v)", latency)
	}
	log.Printf("[NE] Connected successfully (Latency: %v)", latency)

	// 启动读写监听
	go c.conn.WriteListen(func(err error) {
		log.Printf("[NE] Write error: %v", err)
	})
	go c.conn.ReadListen(func(err error) {
		log.Printf("[NE] Read error: %v", err)
	}, c.handleMessage)

	// 启动业务心跳
	go c.appHeartbeatLoop()

	return nil
}

// appHeartbeatLoop 业务心跳循环
func (c *NEClient) appHeartbeatLoop() {
	ticker := time.NewTicker(c.config.HeartbeatInterval)
	defer ticker.Stop()

	// 监听连接关闭信号，一旦断开就退出心跳循环
	closeSig := c.conn.CloseSignal()

	// 立即发送一次
	c.sendHeartbeat()

	for {
		select {
		case <-ticker.C:
			c.sendHeartbeat()
		case <-closeSig:
			return
		case <-c.stopCh:
			return
		}
	}
}

// HeartbeatData 心跳数据包
type HeartbeatData struct {
	NEID      string  `json:"neId"`
	Status    string  `json:"status"`
	Version   string  `json:"version"`
	CPUUsage  float64 `json:"cpuUsage"`
	MemUsage  float64 `json:"memUsage"`
	Timestamp int64   `json:"timestamp"`
}

// sendHeartbeat 发送心跳
func (c *NEClient) sendHeartbeat() {
	// 获取资源信息
	cpuPercent, err := cpu.Percent(0, false)
	cpuVal := 0.0
	if err == nil && len(cpuPercent) > 0 {
		cpuVal = cpuPercent[0]
	}

	vMem, err := mem.VirtualMemory()
	memVal := 0.0
	if err == nil && vMem != nil {
		memVal = vMem.UsedPercent
	}

	data := HeartbeatData{
		NEID:      c.config.NEID,
		Status:    "RUNNING",
		Version:   "1.0.0",
		CPUUsage:  cpuVal,
		MemUsage:  memVal,
		Timestamp: time.Now().UnixMilli(),
	}

	// UUID 生成 (简化)
	uuid := fmt.Sprintf("hb-%d", data.Timestamp)

	// 发送
	c.conn.SendTextJSON(uuid, "heartbeat", data)
	log.Printf("[NE] Heartbeat sent: CPU=%.2f%% Mem=%.2f%%", cpuVal, memVal)
}

// handleMessage 处理服务端下发的消息
func (c *NEClient) handleMessage(conn *ws.ClientConn, msgType int, resp *protocol.Response) {
	// 这里可以处理服务端下发的控制指令
	log.Printf("[NE] Received message: uuid=%s msg=%s code=%d", resp.Uuid, resp.Msg, resp.Code)
}

func main() {
	// 1. 设置日志格式
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// 2. 配置参数
	cfg := Config{
		ServerURL:         "ws://127.0.0.1:33030/ws",
		NEID:              "NE-1001",
		ReconnectInit:     1 * time.Second,
		ReconnectMax:      60 * time.Second,
		HeartbeatInterval: 30 * time.Second,
	}

	// 3. 启动客户端
	client := NewNEClient(cfg)
	log.Println("[System] Starting NE Client...")
	client.Start()

	// 4. 优雅退出
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("[System] Stopping NE Client...")
	client.Stop()
	log.Println("[System] NE Client Stopped")
}
