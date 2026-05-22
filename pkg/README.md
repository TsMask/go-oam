# go-oam/pkg

OAM SDK 核心功能包，可独立引入使用。

## 安装

```bash
go get github.com/tsmask/go-oam/pkg/...
```

## 包一览

| 包 | 说明 | 外部依赖 |
|---|---|---|
| [cmd](#cmd) | 本地命令执行与 Shell 会话 | `creack/pty` |
| [parse](#parse) | 数据解析与格式化 | 无 |
| [crypto](#crypto) | AES 加密、哈希、HMAC | 无 |
| [date](#date) | 日期解析工具 | 无 |
| [generate](#generate) | 随机编码/字符串/数字生成 | 无 |
| [file](#file) | 文件读写、压缩、上传 | 无 |
| [ringbuffer](#ringbuffer) | 泛型环形缓冲区 | 无 |
| [fetch](#fetch) | HTTP 客户端 + 异步推送队列 | `go-resty` |
| [push](#push) | 数据推送领域（告警/KPI/话单等） | 无 |
| [ping](#ping) | ICMP 探测 + 命令行终端 | `pro-bing` |
| [iperf](#iperf) | 网络性能测试命令行终端 | 无 |
| [state](#state) | 系统资源查询（CPU/内存/磁盘/网络/进程） | `gopsutil` |
| [socket](#socket) | TCP/UDP 客户端与服务端 | 无 |
| [telnet](#telnet) | Telnet 客户端与服务端 | 无 |

---

## cmd

本地命令执行与 Shell 会话。

```go
import "github.com/tsmask/go-oam/pkg/cmd"
```

**命令执行**

```go
output, err := cmd.Exec("ls -la")
output, err := cmd.ExecWithTimeOut("ping -c 3 8.8.8.8", 10*time.Second)
output, err := cmd.ExecCommand("ls", "-r", "-l")
```

| 函数 | 说明 |
|---|---|
| `Exec(cmdStr)` | 执行 Shell 命令 |
| `Execf(cmdStr, a...)` | 格式化执行命令 |
| `ExecWithTimeOut(cmdStr, timeout)` | 带超时执行命令 |
| `ExecDirWithTimeOut(workdir, cmdStr, timeout)` | 指定目录带超时执行 |
| `ExecDirScript(workDir, scriptPath)` | 执行脚本文件，默认超时 10 分钟 |
| `ExecCommand(name, a...)` | 执行命令程序带参数 |

**检查工具**

```go
cmd.CheckIllegal("arg1", "arg2")   // 检查特殊字符
cmd.HasNoPasswordSudo()             // 是否有 sudo 权限
cmd.Which("iperf3")                 // 可执行文件是否在 PATH 中
```

**Shell 会话**

```go
session, err := cmd.NewClientSession(80, 24)
defer session.Close()
session.Write("ls\n")
data := session.Read()
session.WindowChange(40, 120)
```

---

## parse

数据解析与格式化，纯 Go 实现。

```go
import "github.com/tsmask/go-oam/pkg/parse"
```

| 函数 | 说明 |
|---|---|
| `Number(value)` | 解析数值，支持 string/int/float/bool，失败返回 0 |
| `Boolean(value)` | 解析布尔值，失败返回 false |
| `Bit(bit)` | 字节数格式化为可读容量字符串（1024 进制），如 `"1.00 KB"` |
| `IsText(data)` | 判断字节切片是否为 UTF-8 文本 |
| `SafeContent(value)` | 敏感内容掩码，首尾可见中间 `*` |
| `ConvertIPMask(bits)` | CIDR 前缀长度转子网掩码，如 `24` → `"255.255.255.0"` |

---

## crypto

AES-CBC 加密、哈希、签名。

```go
import "github.com/tsmask/go-oam/pkg/crypto"
```

**AES 加密**

```go
cipher, _ := crypto.AESEncryptBase64("hello", "1234567890123456")
plain, _ := crypto.AESDecryptBase64(cipher, "1234567890123456")

cipherBytes, _ := crypto.AESEncrypt([]byte("hello"), key)
plainBytes, _ := crypto.AESDecrypt(cipherBytes, key)
```

**哈希**

```go
crypto.MD5("data")                      // hex 编码
crypto.SHA256("data")                   // hex 编码
crypto.HMACSHA256("secret", "data")     // hex 编码
```

---

## date

日期解析工具。

```go
import "github.com/tsmask/go-oam/pkg/date"
```

| 函数 | 说明 |
|---|---|
| `ParseStrToDate(dateStr, formatStr)` | 按格式字符串解析时间 |
| `ParseNumberToDate(dateV)` | 按 Unix 时间戳解析时间 |

---

## generate

随机编码/字符串/数字生成，使用 `crypto/rand` 保证密码学安全。

```go
import "github.com/tsmask/go-oam/pkg/generate"
```

| 函数 | 说明 |
|---|---|
| `Code(size)` | 随机编码（数字+小写字母） |
| `String(size)` | 随机字符串（数字+大小写字母） |
| `Number(size)` | 随机正整数，size 为位数 [1,18] |

---

## file

文件读写、压缩解压、上传分片，纯 Go 实现。

```go
import "github.com/tsmask/go-oam/pkg/file"
```

**CSV**

```go
// 写入（appendMode=true 追加，false 原子覆盖）
file.CSVWrite("data.csv", [][]string{{"name","age"},{"Alice","30"}}, false)

// 流式读取（适合大文件）
header := []string{"name", "age"}
file.CSVRead("data.csv", header, func(row map[string]string) error {
    fmt.Println(row["name"])
    return nil
})

// 全量读取（小文件）
rows, _ := file.CSVReadAll("data.csv")
```

**JSON / JSON Lines**

```go
file.JSONWrite("cfg.json", data)
file.JSONRead("cfg.json", &obj)

file.JSONLineWrite("log.jsonl", []any{obj1, obj2})
file.JSONLineAppend("log.jsonl", newObj)  // 追加一行
file.JSONLineRead("log.jsonl", func(line string) error { ... })
```

**TXT**

```go
file.TXTWrite("note.txt", "hello world")
content, _ := file.TXTRead("note.txt")

// 分隔文本（如 TSV）
file.TXTLineWrite("data.tsv", "\t", [][]string{{"a","b"}})
file.TXTLineRead("data.tsv", "\t", func(fields []string) error { ... })
```

**压缩**

```go
file.TarPack("/data", "/tmp/data.tar")
file.TarUnpack("/tmp/data.tar", "/output")
file.TarGzPack("/data", "/tmp/data.tar.gz")
file.TarGzUnpack("/tmp/data.tar.gz", "/output")

file.ZipPackDir("/data", "/tmp/data.zip")
file.ZipPackFile("/data/file.txt", "/tmp/file.zip")
file.ZipUnpack("/tmp/data.zip", "/output")
```

**文件操作**

```go
file.CopyFile("src.txt", "dst.txt")
file.CopyDir("/src", "/dst")

entries, _ := file.ListDir("/var/log", "*.log")  // 非递归，单 pattern
entry, _ := file.Stat("/var/log/syslog")
exists := file.Exists("/tmp/test.txt")
```

**文件上传**

```go
cfg := file.FileConfig{
    Dir:      "/data/upload",
    MaxSize:  10,              // MB
    AllowExts: []string{".png", ".jpg"},
}

// 单文件保存
upload := file.FileUpload{Name: "photo.png", Data: &imgBytes}
path, err := upload.Save(cfg)

// 分片上传
upload := file.FileUpload{Id: "abc123", Index: 0, Data: &chunk}
upload.ChunkSave(cfg)
files, _ := cfg.ChunkList("abc123")
path, _ := cfg.ChunkMerge("abc123", "photo.png")

// 范围读取（HTTP Range 下载）
fr, _ := file.ReadStream("/data/file.bin", 0, 1024)
defer fr.Close()
```

---

## ringbuffer

泛型环形缓冲区，线程安全。

```go
import "github.com/tsmask/go-oam/pkg/ringbuffer"
```

```go
rb := ringbuffer.NewRingBuffer[string](100)
rb.Push("item")
all := rb.GetAll()
last10 := rb.GetLast(10)
n := rb.Count()
rb.Resize(200)
rb.Clear()
```

---

## fetch

HTTP 客户端（基于 go-resty）+ 异步推送队列。

```go
import "github.com/tsmask/go-oam/pkg/fetch"
```

**同步请求**

```go
opts := fetch.Options{
    Header: map[string]string{"Authorization": "Bearer xxx"},
    Body:   map[string]string{"key": "value"},
    LocalAddr: "10.0.0.1",  // 可选：指定源 IP
}

data, err := fetch.Get("https://api.example.com/items", opts)
data, err := fetch.Post("https://api.example.com/items", opts)
data, err := fetch.Put("https://api.example.com/items/1", opts)
data, err := fetch.Delete("https://api.example.com/items/1", opts)
```

**文件上传**

```go
opts := fetch.Options{
    File: fetch.FileUpload{
        FileName: "report.pdf",
        FieldName: "file",
        Data: &pdfBytes,       // 内存数据
    },
}
data, err := fetch.Post("https://api.example.com/upload", opts)
```

**异步推送**

```go
fetch.AsyncInit(4, 1000)  // 4 协程，队列 1000
defer fetch.AsyncClose()

ctx := context.Background()
fetch.AsyncPush(ctx, "https://api.example.com/event", payload)
```

---

## push

数据推送领域服务，内置环形缓冲区历史管理 + 异步 HTTP 推送。

```go
import "github.com/tsmask/go-oam/pkg/push"
```

| 服务 | 创建 | 推送 URI |
|---|---|---|
| `AlarmService` | `push.NewAlarmService()` | `push.ALARM_PUSH_URI` |
| `KPIService` | `push.NewKPIService(neUid, granularity)` | `push.KPI_PUSH_URI` |
| `NBStateService` | `push.NewNBStateService()` | `push.NB_STATE_PUSH_URI` |
| `UENBService` | `push.NewUENBService()` | `push.UENB_PUSH_URI` |
| `UEMISService` | `push.NewUEMISService()` | `push.UEIMS_PUSH_URI` |
| `CDRService` | `push.NewCDRService()` | `push.CDR_PUSH_URI` |
| `CommonService` | `push.NewCommonService()` | `push.COMMON_PUSH_URI` |

**通用方法**

| 方法 | 说明 |
|---|---|
| `PushURL(url, data, timeout)` | 推送到自定义 URL |
| `HistoryList(n)` | 获取历史记录（n=0 返回全部） |
| `HistorySetSize(size)` | 修改历史记录容量 |

**KPI 额外方法**

```go
kpi := push.NewKPIService("ne-001", 60*time.Second)
kpi.KeySet("cpu", 45.2)
kpi.KeyInc("conn_count")
v := kpi.KeyGet("cpu")
kpi.KPITimerStart(func() string { return "https://api.example.com/kpi" })
// ...
kpi.KPITimerStop()
```

---

## ping

ICMP 探测（基于 pro-bing）+ 命令行终端。

```go
import "github.com/tsmask/go-oam/pkg/ping"
```

**ICMP 探测**

```go
p := ping.Ping{Addr: "8.8.8.8", Count: 4, Timeout: 5 * time.Second}
stats, err := p.Statistics(p)
// stats: map[string]int64{"sent": 4, "recv": 4, "loss": 0, "avg_rtt": 12345}
```

**命令行终端**

```go
t := ping.Terminal{Addr: "8.8.8.8", Count: 4}
cmd, _ := t.ParseOptions()  // 拼装 ping 命令
ver, _ := t.Version()        // 查询 ping 版本
```

---

## iperf

网络性能测试命令行终端。

```go
import "github.com/tsmask/go-oam/pkg/iperf"
```

```go
t := iperf.Terminal{
    Mode:    "client",
    Host:    "10.0.0.1",
    Port:    "5201",
    Time:    10,
    Version: "V3",
}
cmd, _ := t.ParseOptions("V3")  // 拼装 iperf3 命令
ver, _ := t.Version("V3")        // 查询 iperf3 版本
```

---

## state

系统资源查询，合并了进程/网络连接/监控能力。

```go
import "github.com/tsmask/go-oam/pkg/state"
```

**系统信息**

```go
info := state.SystemInfo()      // 主机名、OS、架构、启动时间等
t := state.SystemTime()         // 当前时间
uname := state.StateUName()     // 内核信息
cpu, mem := state.StateProcUsage(pid)  // 进程 CPU/内存
```

**CPU / 内存 / 磁盘**

```go
cpu := state.SystemCPU()        // 型号、主频、各核心使用率
mem := state.SystemMemory()     // 总量、已用、进程占用（字节）
disks := state.SystemDisk(ctx)  // 分区详情（需传入 ctx 控制超时）
```

**网卡**

```go
ifaces := state.SystemNetwork()     // 各网卡 IPv4/IPv6 地址
devices := state.NetworkDevices()   // 前端树形结构（id/label/mac/addrs）
```

**监控采样**

```go
cpuMem := state.LoadCPUMemUsage(3 * time.Second)  // CPU + 内存采样
diskIO := state.LoadDiskIO(3 * time.Second)        // 磁盘 IO 差值
netIO := state.LoadNetIO(3 * time.Second)          // 网卡流量差值
```

**进程查询**

```go
procs, _ := state.Processes(state.PsProcessQuery{
    Name: "nginx",
    User: "root",
})
```

**网络连接查询**

```go
conns, _ := state.NetConnections(state.NetConnectQuery{
    Type: "tcp",
    Port: 80,
})
```

---

## socket

TCP/UDP 客户端与服务端，支持并发安全，`sync.Pool` 复用读缓冲区。

```go
import "github.com/tsmask/go-oam/pkg/socket"
```

**TCP 客户端**

```go
c := &socket.ClientTCP{Addr: "127.0.0.1", Port: "8080"}
c.Connect()
defer c.Close()

// 发送并读取响应，done 返回 true 表示读取完成
resp, err := c.Send([]byte("hello"), 5*time.Second, func(b []byte) bool {
    return bytes.HasSuffix(b, []byte("\n"))
})

// 仅超时
resp, err := c.Send([]byte("ping"), 3*time.Second, nil)
```

**TCP 服务端**

```go
s := &socket.ServerTCP{
    Port:     "8080",
    MaxConns: 100,
    OnError:  func(err error) { log.Println(err) },
}
s.Listen()
defer s.Close()

s.Accept(func(conn net.Conn) {
    buf := make([]byte, 1024)
    n, _ := conn.Read(buf)
    conn.Write(buf[:n])
})
```

**UDP 客户端 / 服务端** — API 与 TCP 一致：

```go
// 客户端
c := &socket.ClientUDP{Addr: "127.0.0.1", Port: "9090"}
c.Connect()
resp, _ := c.Send([]byte("hello"), 3*time.Second, nil)

// 服务端
s := &socket.ServerUDP{Port: "9090", MaxConns: 50}
s.Listen()
s.Serve(func(conn *net.UDPConn, data []byte, addr *net.UDPAddr) {
    conn.WriteToUDP(data, addr)
})
```

---

## telnet

Telnet 客户端与服务端，并发安全。

```go
import "github.com/tsmask/go-oam/pkg/telnet"
```

**客户端**

```go
c := &telnet.Client{Addr: "192.168.1.1", Port: "23"}
if err := c.Connect(); err != nil {
    log.Fatal(err)
}
defer c.Close()

// 可选：登录认证
c.Auth("admin", "password")

// 发送命令，匹配提示符
out, err := c.Send("display version\r\n", 30*time.Second, func(b []byte) bool {
    return bytes.HasSuffix(b, []byte(">")) || bytes.HasSuffix(b, []byte("#"))
})

// 原始读写
c.Write([]byte("enable\r\n"))
data, _ := c.Read(2 * time.Second)
```

**服务端**

```go
s := &telnet.Server{
    Port:     "2323",
    MaxConns: 10,
    OnError:  func(err error) { log.Println(err) },
}
s.Listen()
defer s.Close()

s.Accept(func(conn net.Conn) {
    conn.Write([]byte("Welcome\r\n> "))
    buf := make([]byte, 1024)
    n, _ := conn.Read(buf)
    conn.Write(buf[:n])
})
```

---

## 设计原则

- **并发安全** — 客户端结构体均使用 `sync.RWMutex` + `sync.Mutex` 双锁保护
- **连接池复用** — `fetch` 按源 IP 缓存 HTTP 客户端；`socket`/`telnet` 使用 `sync.Pool` 复用读缓冲区
- **参数注入** — 配置通过结构体传入
- **`done func([]byte) bool`** — 统一的读取终止回调模式（socket/telnet）
