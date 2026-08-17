# go-oam/pkg

OAM SDK 的通用工具包集合。每个子包都可以单独引入，包内 API 以当前源码为准。

## 安装

```bash
go get github.com/tsmask/go-oam
```

## 包一览

| 包 | 说明 | 主要外部依赖 |
|---|---|---|
| `pkg/cmd` | 本地命令执行、检查工具和 Bash PTY 会话 | `creack/pty` |
| `pkg/parse` | 数值、布尔、容量、掩码和文本判断 | 无 |
| `pkg/crypto` | AES-CBC、MD5、SHA-256、HMAC-SHA256 | 标准库 |
| `pkg/date` | 字符串和数字时间戳解析 | 标准库 |
| `pkg/generate` | 安全随机编码、字符串和数字 | 标准库 `crypto/rand` |
| `pkg/file` | 文件读写、上传、复制、tar/zip | 标准库 |
| `pkg/ringbuffer` | 泛型线程安全环形缓冲区 | 标准库 |
| `pkg/fetch` | HTTP 请求封装和全局异步 POST 队列 | `go-resty` |
| `pkg/state` | CPU、内存、磁盘、网络、进程和连接采集 | `gopsutil` |
| `pkg/socket` | TCP/UDP 客户端和服务端 | 标准库 |
| `pkg/telnet` | Telnet 客户端和服务端 | 标准库 |
| `pkg/ssh` | SSH、SFTP 和远程交互式终端 | `golang.org/x/crypto/ssh`、`github.com/pkg/sftp` |

---

## cmd

`cmd` 提供本地命令执行和工具检查。Shell 命令在 Windows 使用 `powershell -Command`，在其他平台使用 `bash -c`；`ExecCommand` 不经过 shell，直接执行程序和参数。

```go
import "github.com/tsmask/go-oam/pkg/cmd"
```

| 函数 | 说明 |
|---|---|
| `Exec(cmdStr)` | 执行 shell 命令，成功返回 stdout |
| `Execf(cmdStr, a...)` | 格式化后执行 shell 命令 |
| `ExecWithTimeOut(cmdStr, timeout)` | 带 context 超时的 shell 命令 |
| `ExecDirWithTimeOut(workdir, cmdStr, timeout)` | 指定工作目录和超时 |
| `ExecDirScript(workdir, scriptPath)` | Windows 执行 `powershell -File`，其他平台执行 `bash script`；固定 10 分钟超时 |
| `ExecCommand(name, a...)` | 直接执行程序，不经过 shell |

命令失败时，`runCmd` 会把已捕获的 stderr/stdout 拼进返回的 error 字符串，并同时返回原始执行错误。超时判断基于 `context.DeadlineExceeded`，超时返回 `errCmdTimeout ...` 包装错误。

检查函数：

```go
hasIllegal := cmd.CheckIllegal("arg1", "arg2")
hasSudo := cmd.HasNoPasswordSudo()
sudoPrefix := cmd.SudoHandleCmd() // 有免密 sudo 时为 "sudo "
found := cmd.Which("bash")
```

`CheckIllegal` 只检查 `&`、`|`、`;`、`$`、单引号、反引号、圆括号和双引号，返回 `true` 表示包含这些字符；它不负责转义或拦截命令。

PTY 会话固定启动 `bash`，主要面向 Unix 环境。`NewClientSession(cols, rows)` 的参数是列和行；写入时需要自行带换行符，读取是阻塞读且每次最多返回 4096 字节。

```go
session, err := cmd.NewClientSession(120, 40)
if err != nil {
    return err
}
defer session.Close()

_, _ = session.Write("uname -a\n")
data := session.Read()
_ = session.WindowChange(180, 50)
```

## parse

```go
import "github.com/tsmask/go-oam/pkg/parse"
```

| 函数 | 实际行为 |
|---|---|
| `IsText(data)` | 空切片视为文本；要求 UTF-8 有效，且除 `\n`、`\r`、`\t` 外不允许控制字符 |
| `Number(value)` | 支持 string、整数、无符号整数、float、bool；字符串按十进制 int64 解析，float 截断，失败返回 0 |
| `Boolean(value)` | 字符串使用 `strconv.ParseBool`，数值类型按非零判断，失败返回 false |
| `Bit(bit)` | 1024 进制容量格式化，保留两位小数，单位到 YB |
| `SafeContent(value)` | 按 rune 长度分级掩码；空串返回空串，不会越界 |
| `ConvertIPMask(bits)` | CIDR 前缀长度转点分掩码；小于 0 或大于 32 返回 `255.255.255.255` |

```go
n := parse.Number("42")
b := parse.Boolean("true")
size := parse.Bit(1536)       // "1.50 KB"
masked := parse.SafeContent("secret")
mask := parse.ConvertIPMask(20) // "255.255.240.0"
```

## crypto

```go
import "github.com/tsmask/go-oam/pkg/crypto"
```

AES 使用 CBC 模式、PKCS#7 填充和随机 IV，密文结构为 `IV || ciphertext`。Base64 版本处理空字符串时直接返回空字符串和 nil。

```go
ciphertext, err := crypto.AESEncryptBase64("hello", "1234567890123456")
plaintext, err := crypto.AESDecryptBase64(ciphertext, "1234567890123456")

raw, err := crypto.AESEncrypt([]byte("hello"), []byte("1234567890123456"))
plain, err := crypto.AESDecrypt(raw, []byte("1234567890123456"))
```

key 长度必须是 16、24 或 32 字节。解密会校验密文长度、块对齐和填充完整性。

哈希函数：

```go
md5Sum := crypto.MD5("data")
sha256Sum := crypto.SHA256("data")
hmacSum := crypto.HMACSHA256("secret", "data")
```

三者均返回小写 hex 字符串。MD5 只适用于非安全场景的摘要。

## date

```go
import "github.com/tsmask/go-oam/pkg/date"
```

`ParseStrToDate(dateStr, formatStr)` 默认格式为 `time.DateTime`，使用本地时区解析；空串、`<nil>` 和非法输入返回零值 `time.Time{}`。

`ParseNumberToDate(dateV)` 根据数值范围推断单位：

- `dateV > 1e15`：微秒
- `dateV > 1e12`：毫秒
- `dateV > 1e9`：秒
- `0`、等于边界值或其他不识别范围：零值时间

```go
t1 := date.ParseStrToDate("2026-08-17 10:00:00", "")
t2 := date.ParseNumberToDate(1770000000000)
```

## generate

```go
import "github.com/tsmask/go-oam/pkg/generate"
```

| 函数 | 实际行为 |
|---|---|
| `Code(size)` | 数字和小写字母随机串；`size <= 0` 返回空串 |
| `String(size)` | 数字、大小写字母随机串；`size <= 0` 返回空串 |
| `Number(size)` | 随机固定位数正整数；范围限制为 1 到 18 位，首位非零 |

随机源为 `crypto/rand`。如果系统随机源故障，字符串函数会退化为字母表首字符，数字函数会返回该位数的最小值。

## file

```go
import "github.com/tsmask/go-oam/pkg/file"
```

### 结构化文件

```go
_ = file.CSVWrite("data.csv", [][]string{
    {"name", "age"},
    {"Alice", "30"},
}, false)

_ = file.CSVRead("data.csv", nil, func(row map[string]string) error {
    fmt.Println(row["name"])
    return nil
})

rows, _ := file.CSVReadAll("data.csv")
```

`CSVWrite` 的 `appendMode=false` 使用唯一临时文件加 rename 的覆盖写入；`appendMode=true` 直接追加并调用 `Sync`。`CSVRead` 传 nil 表头时自动把首行转小写并去空白；传非 nil 表头会跳过首行。CSV 读取允许 LazyQuotes，且不强制每行字段数一致。

JSON 和 JSON Lines：

```go
_ = file.JSONWrite("cfg.json", data)
_ = file.JSONRead("cfg.json", &obj)

_ = file.JSONLineWrite("log.jsonl", []any{obj1, obj2})
_ = file.JSONLineAppend("log.jsonl", obj3)
_ = file.JSONLineRead("log.jsonl", func(line string) error { return nil })
lines, _ := file.JSONLineReadAll("log.jsonl")
```

JSON/JSONL 的覆盖写入是原子写入；JSONL 追加会按同路径互斥并调用 `Sync`，但异常断电等系统级故障仍可能留下部分尾部。JSONL 每行最大扫描长度为 1MB。

分隔文本：

```go
_ = file.TXTWrite("note.txt", "hello")
content, _ := file.TXTRead("note.txt")

_ = file.TXTLineWrite("data.tsv", "\t", [][]string{{"a", "b"}})
fields, _ := file.TXTLineReadAll("data.tsv", "\t")
```

这些函数的“线程安全”指同一进程内按规范化后的相同路径互斥，覆盖写入和读取也不会交错；不同路径可并发，少量哈希碰撞会额外串行化。流式读取回调执行期间会持有该路径锁，回调内不要再操作同一路径。该锁不提供跨进程文件锁。

### 文件与目录信息

```go
entries, _ := file.ListDir("/var/log", "*.log")
entry, _ := file.Stat("/var/log/syslog")
exists := file.Exists("/tmp/test.txt")

_ = file.CopyFile("src.txt", "dst.txt")
_ = file.CopyDir("/src", "/dst")
```

`ListDir` 只遍历当前目录，按修改时间倒序；无效 glob 表达式会返回错误。`CopyFile` 只接受常规源文件，保留源文件权限，通过唯一临时文件原子替换目标；源和目标是同一个文件时直接跳过。`CopyDir` 递归复制目录。

### 上传与分片

```go
cfg := file.FileConfig{
    Dir:        "/data/upload",
    MaxSize:    10,
    AllowExts:  []string{".png", ".jpg"},
    MaxNameLen: 100,
}

data := []byte("png bytes")
path, err := (file.FileUpload{
    Name: "photo.png",
    Data: &data,
}).Save(cfg)
```

`MaxSize` 单位是 MB，小于 1 时按 1MB 处理，作用于单个文件或单个分片，不限制分片合并后的总大小。`Reader` 优先于 `Data`，流式写入临时文件并限制读取长度。路径输入推荐统一使用 `/`；`Save` 和 `ChunkMerge` 成功后返回清理过的 `/` 分隔路径。Windows 额外接受 `\`，但 Unix 中 `\` 是普通文件名字符。

`Save` 要求提供 `Name` 和 `Reader`/`Data` 其中之一；`ChunkSave` 要求 `Id`、`Index` 和 `Reader`/`Data`。上传文件名中的路径分隔符和 Windows 非法字符会被清理为 `_`，不会拼入客户端提交的目录。分片 `Id` 只能是安全的单段文件名，`Index` 必须是非负十进制整数。同一存储目录内的上传和分片操作会进程内互斥。

分片上传：

```go
chunk := []byte("chunk")
upload := file.FileUpload{Id: "abc123", Index: "0", Data: &chunk}
_ = upload.ChunkSave(cfg)

names, _ := cfg.ChunkList("abc123")
path, err := cfg.ChunkMerge("abc123", "photo.png")
```

`ChunkList` 按分片序号返回。`ChunkMerge` 会校验最终文件名、长度和 `AllowExts`，先写入唯一临时文件，成功替换后才删除分片目录；合并失败会保留分片且不留下不完整输出。

范围读取：

```go
r, err := file.ReadStream("/data/file.bin", 0, 1023)
```

虽然函数名为 `ReadStream`，当前实现会把请求范围完整读入 `FileRange.Data`，适合小范围读取。`end <= 0` 表示读到文件末尾；空文件返回 `End=-1` 和空数据。

### 归档

```go
_ = file.TarPack("/data", "/tmp/data.tar")
_ = file.TarUnpack("/tmp/data.tar", "/output")
_ = file.TarGzPack("/data", "/tmp/data.tar.gz")
_ = file.TarUnpack("/tmp/data.tar.gz", "/output")

_ = file.ZipPackDir("/data", "/tmp/data.zip")
_ = file.ZipPackFile("/data/file.txt", "/tmp/file.zip")
_ = file.ZipUnpack("/tmp/data.zip", "/output")
```

tar/tar.gz 使用 `gzip.BestSpeed` 压缩，zip 使用 Deflate。归档输出通过唯一临时文件原子替换。解包使用 `filepath.Rel` 判断目标边界，`.` 等相对输出目录可用，试图逃逸输出目录的条目会返回错误；tar/zip 只落盘普通文件和目录，非常规文件不会被还原。

## ringbuffer

```go
import "github.com/tsmask/go-oam/pkg/ringbuffer"

rb := ringbuffer.NewRingBuffer[string](100)
rb.Push("item")
all := rb.GetAll()
last10 := rb.GetLast(10)
n := rb.Count()
_ = rb.Resize(200)
rb.Clear()
```

`size <= 0` 时默认 1024。满员后 `Push` 覆盖最旧元素。`GetAll` 和 `GetLast` 都按插入顺序返回；`Resize` 缩小时保留最新元素，`newSize <= 0` 时忽略本次调整。

## fetch

`fetch` 基于 `go-resty` 封装 HTTP 请求。请求方法返回响应 body；HTTP 4xx/5xx 时同时返回 body 和 `HTTP <status>` 错误。

```go
import "github.com/tsmask/go-oam/pkg/fetch"

opts := fetch.Options{
    Ctx:     ctx,
    Headers: map[string]string{"Authorization": "Bearer token"},
    Query:   map[string]string{"page": "1"},
    JSON: map[string]any{
        "key": "value",
    },
}

data, err := fetch.Get("https://api.example.com/items", opts)
data, err = fetch.Post("https://api.example.com/items", opts)
data, err = fetch.Put("https://api.example.com/items/1", opts)
data, err = fetch.Delete("https://api.example.com/items/1", opts)
data, err = fetch.Request(http.MethodPatch, "https://api.example.com/items/1", opts)
```

`Options` 的主要字段：

| 字段 | 说明 |
|---|---|
| `Ctx` | 请求 context，用于取消和超时 |
| `Headers` / `Query` | 请求头和查询参数 |
| `JSON` / `Form` | JSON body 或表单 body；不建议同时设置 |
| `Files` | multipart 文件列表 |
| `Debug` | 开启 resty 调试日志 |
| `LocalAddr` | 绑定源 IP；无效 IP 会被忽略并使用默认路由 |
| `RetryCount` | resty 客户端重试次数，默认 0 |
| `RetryWaitTime` / `RetryMaxWait` | 重试等待，默认分别为 300ms 和 5s |

文件上传：

```go
pdf := []byte("%PDF-...")
opts := fetch.Options{
    Files: []fetch.FileUpload{{
        Field:  "file",
        Name:   "report.pdf",
        Data:   &pdf,
    }},
}
data, err := fetch.Post("https://api.example.com/upload", opts)
```

`FileUpload` 的选择顺序是 `Reader`、`Data`、`Path`。使用 Reader/Data 时必须提供 `Name`；使用 Path 时直接读取本地文件。

HTTP client 按 `LocalAddr` 和重试配置缓存复用，固定客户端超时为 1 分钟；`Options.Ctx` 仍可用于提前取消。重试只会在配置 `RetryCount > 0` 时启用。

### 异步 POST 队列

```go
fetch.AsyncInit(4, 1000)
defer fetch.AsyncClose()

err := fetch.AsyncPush(ctx, "https://api.example.com/event", map[string]any{"k": "v"})
```

`AsyncInit` 只生效一次，未调用时 `AsyncPush` 会使用默认 2 个 worker、500 队列容量。队列未满时入队即返回 nil；队列满时降级为同步 POST，只有这个同步降级失败才会返回错误。worker 后续处理失败只通过标准 log 输出，不会回调调用方。

`AsyncClose` 会关闭队列并等待所有 worker 排空退出。当前全局队列关闭后不能重新初始化，因此应在进程退出前调用；关闭后继续 `AsyncPush` 不是有效用法。

## state

```go
import "github.com/tsmask/go-oam/pkg/state"
```

系统基础信息：

```go
info := state.SystemInfo()
t := state.SystemTime()
uname := state.StateUName()
cpuUsage, memUsage := state.StateProcUsage(1234)
```

`SysInfo.BootTime` 实际是系统已运行秒数，`RunTime` 是当前 Go 进程已运行秒数。`StateUName` 在 Linux/macOS 执行 `uname -a`，Windows 走 gopsutil 的主机信息。`StateProcUsage` 采集系统 CPU 时会阻塞约 200ms。

资源快照：

```go
cpu := state.SystemCPU()             // 阻塞约 200ms
memory := state.SystemMemory()
disks := state.SystemDisk(ctx)       // ctx 控制分区和使用率采集
ifaces := state.SystemNetwork()
devices := state.NetworkDevices()
```

`SystemDisk` 返回 nil 表示分区枚举失败；单个分区使用率失败会被跳过。网络接口查询跳过回环接口和无地址接口，但保留链路本地地址。

周期差值采样：

```go
cpuMem := state.LoadCPUMemUsage(3 * time.Second)
diskIO := state.LoadDiskIO(3 * time.Second)
netIO := state.LoadNetIO(3 * time.Second)
```

`LoadCPUMemUsage` 会在 CPU 采样窗口内阻塞；Linux/macOS 额外读取 load，Windows 上 load 获取失败时只返回可用字段。`LoadDiskIO` 和 `LoadNetIO` 都是“首快照、sleep duration、末快照、计算差值”，结果包含虚拟网卡；两块网卡计数器回退时不产生负数。

进程和网络连接：

```go
procs, err := state.Processes(state.PsProcessQuery{
    Name:     "nginx",
    Username: "root",
})

conns, _ := state.NetConnections(state.NetConnectQuery{
    Port: 80,
    Name: "nginx",
    PID:  1234,
})
```

进程查询条件之间是 AND 关系，`Name` 和 `Username` 为 contains 模糊匹配，结果按 PID 升序；内部用 4 个 goroutine 并发采集，部分字段采集失败会保留零值。

`NetConnections` 当前没有协议过滤字段，始终遍历 tcp 和 udp，`Port` 匹配本地或远端端口。进程已退出或进程名读取失败的连接会被跳过；单个协议枚举失败会被忽略。

## ssh

```go
import "github.com/tsmask/go-oam/pkg/ssh"

client, err := ssh.New(
    ssh.WithHost("192.168.1.10", 22),
    ssh.WithUser("root"),
    ssh.WithPassword("password"),
    ssh.WithDialTimeout(5*time.Second),
    ssh.WithKeepAlive(30*time.Second),
)
if err != nil {
    return err
}
defer client.Close()
```

默认端口 22、用户 root、拨号超时 5 秒、keepalive 关闭。必须提供密码或私钥其中一种认证方式。`WithPrivateKey(privateKey, passPhrase)` 接收 PEM 字符串，不是文件路径。

当前实现使用 `ssh.InsecureIgnoreHostKey()`，不校验远程主机密钥。不要在不受信任网络中把它作为唯一安全边界。

远程命令：

```go
output, err := client.Exec("uname -a")
```

`Exec` 每次创建一个 SSH session，返回 stdout 和 stderr 的合并结果；命令执行失败时可能同时返回非空 output 和 error。

交互式终端：

```go
session, err := client.NewSession(120, 40)
if err != nil {
    return err
}
defer session.Close()

_, _ = session.Write("uptime\n")
data := session.Read()
_ = session.WindowChange(180, 50)
```

`NewSession(cols, rows)` 请求 `xterm` PTY 并启动 shell。`Read` 阻塞等待第一块输出，再排空当前 channel 中已到达的数据；它没有超时参数，需要调用方自行控制生命周期。`Write` 只写原始字符串，换行符由调用方提供。

SFTP：

```go
sftp, err := client.NewSFTP()
if err != nil {
    return err
}
defer sftp.Close()

entries, _ := sftp.ListDir("/var/log", "*.log")
info, _ := sftp.Stat("/var/log/syslog")
exists := sftp.Exists("/var/log/syslog")

_ = sftp.Upload("/tmp/local.txt", "/tmp/remote.txt")
_ = sftp.Download("/tmp/remote.txt", "/tmp/local.txt")
_ = sftp.UploadDir("/local/dir", "/remote/dir")
_ = sftp.DownloadDir("/remote/dir", "/local/dir")
_ = sftp.RemoveOldFiles("/remote/dir", time.Now().Add(-24*time.Hour))
```

`ListDir` 的 pattern 使用 `filepath.Match`，结果按修改时间倒序。单文件上传会自动创建远程父目录，下载会创建本地父目录。目录递归传输遇到符号链接时按 filepath.Walk 的行为处理。`DownloadDir` 对单个文件或子目录失败采用继续处理策略，最终仍可能返回 nil；需要严格错误处理时应自行遍历或检查结果。`RemoveOldFiles` 只处理普通文件，单个删除失败会跳过。

启用 keepalive 后，客户端按间隔发送 `keepalive@openssh.com` 全局请求；发送失败会主动关闭客户端连接。

## socket

`socket` 提供传输层原语，不做协议拆包。客户端由后台 readLoop 读取数据并写入 channel，`Read()` 阻塞返回一个数据块或一个 UDP datagram。

### TCP 客户端

```go
c := &socket.ClientTCP{
    Addr:         "127.0.0.1",
    Port:         "9090",
    DialTimeout:  5 * time.Second,
    ReadTimeout:  10 * time.Second,
    WriteTimeout: 5 * time.Second,
    TCPKeepAlive: 30 * time.Second,
    Context:      ctx,
}
if err := c.Connect(); err != nil {
    return err
}
defer c.Close()

if _, err := c.Write([]byte("ping\n")); err != nil {
    return err
}
data, err := c.Read()
```

| 字段/方法 | 说明 |
|---|---|
| `Addr` / `Port` | 必填 |
| `DialTimeout` | 0 表示 10s |
| `ReadTimeout` / `WriteTimeout` | 底层单次 IO deadline；0 表示不限 |
| `TCPKeepAlive` | 0 使用系统默认 |
| `Context` | 取消时联动关闭连接 |
| `Connect()` | 已连接时幂等返回 nil；关闭后返回 `ErrClientClosed`；拨号失败后仍可重试 |
| `Write(data)` | 内部串行化，可并发调用 |
| `Read()` | 返回一个读取块；连接结束后返回 `io.EOF` |
| `Close()` | 幂等，唤醒阻塞 Read；未消费的缓冲数据会被丢弃 |
| `State()` / `IsConnected()` | 状态观测 |
| `OnError(fn)` | 远端断开或读循环异常时回调；用户主动 Close 不触发 |

`Read()` 没有内置调用方超时。需要“读直到某标志”的协议逻辑，应由业务层在外层用 `select`、`Context` 或定时器实现。

### TCP 服务端

```go
s := &socket.ServerTCP{
    Handler: func(c *socket.Conn) error {
        buf := make([]byte, 4096)
        n, err := c.Read(buf)
        if err != nil {
            return err
        }
        _, err = c.Write(buf[:n])
        return err
    },
    OnError:      func(err error) { log.Println(err) },
    MaxConns:     100,
    TCPKeepAlive: 30 * time.Second,
    Context:      ctx,
}

go func() {
    if err := s.Listen(":9090"); err != nil && !errors.Is(err, socket.ErrServerClosed) {
        log.Println(err)
    }
}()
defer s.Close()
```

`Listen(address)` 阻塞服务，地址格式与 `net.Listen` 相同。`Close` 阻塞等待所有 handler goroutine 退出。`Conn` 提供 `Read`、`Write`、`Close`、`Context`、`Server`、`RemoteAddr`、`LocalAddr` 和 `IsClosed`；底层 `net.TCPConn` 的并发写不由 `Conn` 串行化，多个 goroutine 写同一连接时必须业务层加锁。

### UDP

```go
// 客户端
c := &socket.ClientUDP{
    Addr:        "127.0.0.1",
    Port:        "9091",
    DialTimeout: 5 * time.Second,
}
if err := c.Connect(); err != nil {
    return err
}
defer c.Close()

_, _ = c.Write([]byte("ping"))
datagram, _ := c.Read()

// 服务端
s := &socket.ServerUDP{
    Handler: func(pc *socket.PacketConn, data []byte, addr *net.UDPAddr) error {
        _, err := pc.WriteToUDP(data, addr)
        return err
    },
    OnError:  func(err error) { log.Println(err) },
    MaxConns: 100,
    Context:  ctx,
}
go func() { _ = s.Listen(":9091") }()
defer s.Close()
```

UDP 客户端 `Read` 一次返回一个完整 datagram，不做流式拼装；同地址回包可能乱序。服务端每个 datagram 调用一次 handler，`MaxConns` 限制的是并发 handler 数，达到上限时丢包并通过 `OnError` 通知。`PacketConn` 在 handler 返回后自动释放，不能跨 handler 保存使用。

`socket` 导出的错误包括 `ErrClientClosed`、`ErrClientNotConnected`、`ErrServerClosed`、`ErrServerNotStarted`、`ErrAlreadyServing` 和 `ErrNoHandler`。

## telnet

### 客户端

```go
c := &telnet.Client{
    Addr:           "192.168.1.1",
    Port:           "23",
    DialTimeout:    5 * time.Second,
    TCPKeepAlive:   30 * time.Second,
    MaxRead:        1 << 20,
    AuthPromptWait: 2 * time.Second,
}
if err := c.Connect(); err != nil {
    return err
}
defer c.Close()

c.OnError(func(err error) { log.Println("telnet:", err) })

if err := c.Auth("admin", "password"); err != nil {
    return err
}

out, err := c.Exec("display version\r\n", func(b []byte) bool {
    return bytes.HasSuffix(b, []byte(">")) || bytes.HasSuffix(b, []byte("#"))
})
```

| 字段/方法 | 说明 |
|---|---|
| `Addr` / `Port` | 必填 |
| `DialTimeout` | 0 表示 10s |
| `ReadTimeout` / `WriteTimeout` | 底层单次 IO deadline；0 表示不限 |
| `TCPKeepAlive` | 0 使用系统默认 |
| `MaxRead` | `Exec` 最大读取字节数；0 表示 1MB，超出返回 `ErrClientTruncated` |
| `AuthPromptWait` | Auth 等待 prompt 的时间；0 表示 2s，超时后仍会继续发送凭据 |
| `Newline` | Auth 凭据行尾；空串表示 `\r\n` |
| `KeepIAC` | `Exec` 是否保留 Telnet IAC 协商字节；默认过滤 |
| `Exec(cmd, done)` | 写命令并读直到 done 命中或连接关闭；done 为 nil 时读到关闭 |
| `Auth(user, password)` | 空用户名或密码会跳过对应步骤，两者都为空直接返回 nil |
| `WindowChange(h, w)` | 发送 NAWS 窗口变更，参数是行和列 |

`Auth` 的 prompt 等待当前只消费一个读取块；prompt 跨 TCP 分段时可能错位。客户端关闭后不可复用；用户主动 `Close` 不触发 `OnError`，远端断开和读循环异常会触发。

并发约束是无锁约定：`Exec`、`Auth` 和直接 `Read` 不要并发调用，否则命令字节流或输出消费者会交错；`Write` 自身串行化，`Close` 可随时调用并唤醒阻塞读。

### 服务端

```go
s := &telnet.Server{
    Handler: func(c *telnet.Conn) error {
        if _, err := c.Write([]byte("login: ")); err != nil {
            return err
        }
        buf := make([]byte, 1024)
        n, err := c.Read(buf)
        if err != nil {
            return err
        }
        _, err = c.Write(buf[:n])
        return err
    },
    OnError:      func(err error) { log.Println(err) },
    MaxConns:     100,
    TCPKeepAlive: 30 * time.Second,
}

go func() {
    if err := s.Listen(":2323"); err != nil && !errors.Is(err, telnet.ErrServerClosed) {
        log.Println(err)
    }
}()
defer s.Close()
```

服务端是原始 Telnet TCP 服务，不主动协商认证或终端选项。Handler 返回 error 或 panic 会统一交给 `OnError`；达到 `MaxConns` 时先写提示再断开。`Close` 幂等，并阻塞到所有 handler goroutine 退出，期间通过连接 Context 和 deadline 唤醒阻塞 IO。

`telnet.Conn` 提供 `Read`、`Write`、`Close`、`Context`、`Server`、`RemoteAddr`、`LocalAddr` 和 `IsClosed`。错误包括 `ErrClientClosed`、`ErrClientNotConnected`、`ErrClientTruncated`、`ErrServerClosed`、`ErrAlreadyServing` 和 `ErrNoHandler`。

---

## 设计与验证

- 工具包优先暴露小型函数和结构体字段配置；生命周期对象必须在 `Connect`、`Listen` 或 `New` 之前完成配置。
- `socket` 和 `telnet` 客户端关闭后不可复用，重连需要创建新对象；服务端 `Close` 后再次 `Listen` 返回 `ErrServerClosed`。
- `fetch` 的异步能力是进程内队列，不提供持久化、跨进程投递或投递确认。
- `state` 的采样函数多为阻塞式，接入长期监控循环时建议放在独立 goroutine 或任务调度器中。
- `ssh` 当前不校验主机密钥，SFTP 目录下载是尽力而为语义，生产使用前应评估安全性和错误策略。

```bash
go test ./pkg/...
go vet ./pkg/...
```
