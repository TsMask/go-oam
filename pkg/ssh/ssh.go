package ssh

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

// Client SSH客户端
//
// 支持密码和私钥认证，提供远程命令执行、文件列表、交互式会话和SFTP文件传输。
//
// 示例:
//
//	client, err := ssh.New(
//	    ssh.WithHost("192.168.1.1", 22),
//	    ssh.WithUser("root"),
//	    ssh.WithPassword("password"),
//	)
//	defer client.Close()
//	output, err := client.RunCMD("ls -la")
type Client struct {
	addr        string
	port        int
	user        string
	password    string
	privateKey  string
	passPhrase  string
	dialTimeout time.Duration

	sshClient *gossh.Client
	closed    atomic.Bool
	closeOnce sync.Once
}

// New 创建SSH客户端并建立连接
func New(opts ...Option) (*Client, error) {
	c := &Client{
		port:        22,
		user:        "root",
		dialTimeout: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.addr == "" {
		return nil, fmt.Errorf("ssh host address not set")
	}
	if c.password == "" && c.privateKey == "" {
		return nil, fmt.Errorf("ssh host authentication method not set")
	}
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c, nil
}

// connect 连接管理
func (c *Client) connect() error {
	dialAddr := net.JoinHostPort(c.addr, strconv.Itoa(c.port))

	config := &gossh.ClientConfig{
		User:            c.user,
		Timeout:         c.dialTimeout,
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
	}
	config.SetDefaults()

	if c.privateKey != "" {
		var signer gossh.Signer
		var err error
		if c.passPhrase != "" {
			signer, err = gossh.ParsePrivateKeyWithPassphrase([]byte(c.privateKey), []byte(c.passPhrase))
		} else {
			signer, err = gossh.ParsePrivateKey([]byte(c.privateKey))
		}
		if err != nil {
			return err
		}
		config.Auth = []gossh.AuthMethod{gossh.PublicKeys(signer)}
	} else {
		config.Auth = []gossh.AuthMethod{gossh.Password(c.password)}
	}

	client, err := gossh.Dial("tcp", dialAddr, config)
	if err != nil {
		return err
	}
	c.sshClient = client
	return nil
}

// Close 关闭SSH连接
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		if c.sshClient != nil {
			c.sshClient.Close()
		}
	})
}

// Addr 返回主机地址
func (c *Client) Addr() string { return c.addr }

// User 返回登录用户名
func (c *Client) User() string { return c.user }

// Exec 执行远程命令，返回标准输出和错误输出的合并结果
func (c *Client) Exec(cmdStr string) (string, error) {
	if c.closed.Load() {
		return "", fmt.Errorf("ssh was closed")
	}
	session, err := c.sshClient.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	buf, err := session.CombinedOutput(cmdStr)
	return string(buf), err
}
