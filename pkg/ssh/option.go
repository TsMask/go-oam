package ssh

import "time"

// Option SSH客户端配置选项
type Option func(*Client)

// WithHost 设置主机地址和端口，默认端口 22
func WithHost(addr string, port int) Option {
	return func(c *Client) {
		if addr != "" {
			c.addr = addr
		}
		if port > 0 {
			c.port = port
		}
	}
}

// WithUser 设置登录用户名，默认 "root"
func WithUser(user string) Option {
	return func(c *Client) {
		if user != "" {
			c.user = user
		}
	}
}

// WithPassword 设置密码认证
func WithPassword(password string) Option {
	return func(c *Client) {
		if password != "" {
			c.password = password
		}
	}
}

// WithPrivateKey 设置私钥认证，passPhrase 为私钥密码可为空
func WithPrivateKey(privateKey, passPhrase string) Option {
	return func(c *Client) {
		if privateKey != "" {
			c.privateKey = privateKey
			c.passPhrase = passPhrase
		}
	}
}

// WithDialTimeout 设置连接超时，默认 5s
func WithDialTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.dialTimeout = d
		}
	}
}

// WithKeepAlive 设置心跳间隔，<=0 表示禁用（默认禁用）。
// 启用后客户端会按此间隔向服务端发送 keepalive 探活，
// 防止被服务端 idle 策略踢出连接。
func WithKeepAlive(interval time.Duration) Option {
	return func(c *Client) {
		if interval > 0 {
			c.keepAlive = interval
		}
	}
}
