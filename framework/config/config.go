package config

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"dario.cat/mergo"
	"gopkg.in/yaml.v3"
)

//go:embed config.yaml
var configFile embed.FS

// DefaultConfigFileName 加载当前目录下的默认配置文件
const DefaultConfigFileName = "oam.yaml"

// Config 顶级配置结构体
type Config struct {
	Route     []RouteConfig `yaml:"route" json:"route"`
	NE        NEConfig      `yaml:"ne" json:"ne"`
	OMC       OMCConfig     `yaml:"omc" json:"omc"`
	Upload    UploadConfig  `yaml:"upload" json:"upload"`
	mu        sync.RWMutex  `yaml:"-" json:"-"` // 配置读写锁
	startTime time.Time     `yaml:"-" json:"-"` // 启动时间
}

// RouteConfig 路由配置
type RouteConfig struct {
	Addr   string `yaml:"addr" json:"addr"`     // 监听地址 格式：ip:port
	Schema string `yaml:"schema" json:"schema"` // 协议 http/https
	Cert   string `yaml:"cert" json:"cert"`     // 证书文件路径，仅https协议需要
	Key    string `yaml:"key" json:"key"`       // 私钥文件路径，仅https协议需要
}

// NEConfig 网元信息配置
type NEConfig struct {
	Type       string `yaml:"type" json:"type"`             // 网元类型 大写
	Version    string `yaml:"version" json:"version"`       // 版本号 格式：X.Y.Z
	SerialNum  string `yaml:"serialNum" json:"serialNum"`   // 序列号 8位字符
	ExpiryDate string `yaml:"expiryDate" json:"expiryDate"` // 有效日期 格式：YYYY-MM-DD
	NbNumber   int    `yaml:"nbNumber" json:"nbNumber"`     // 基站限制数量 AMF MME
	UeNumber   int    `yaml:"ueNumber" json:"ueNumber"`     // 终端限制数量 UDM
	Pid        int    `yaml:"pid" json:"pid"`               // 进程ID 外部程序运行时需要填，不填默认当前
}

// OMCConfig 网管信息配置
type OMCConfig struct {
	URL            string `yaml:"url" json:"url"`                       // 网管地址 如：http://127.0.0.1:5678
	NeUID          string `yaml:"neUID" json:"neUID"`                   // 网元唯一标识 如：12345678
	CoreUID        string `yaml:"coreUID" json:"coreUID"`               // 核心网唯一标识 12345678
	KPIGranularity int    `yaml:"kpiGranularity" json:"kpiGranularity"` // KPI 采集粒度 单位秒
}

// UploadConfig 文件上传配置
type UploadConfig struct {
	FileDir   string   `yaml:"fileDir" json:"fileDir"`
	FileSize  int      `yaml:"fileSize" json:"fileSize"`
	WhiteList []string `yaml:"whiteList" json:"whiteList"`
}

// New 创建一个新的配置实例
func New() *Config {
	c := &Config{
		startTime: time.Now(),
	}
	// 1. 加载内置配置
	c.InitFromEmbed()

	// 2. 尝试加载当前目录下的默认配置文件
	if _, err := os.Stat(DefaultConfigFileName); err == nil {
		extC, err := LoadExternalConfig(DefaultConfigFileName)
		if err == nil {
			c.Merge(extC)
		}
	}

	return c
}

// InitFromEmbed 从内置的 config.yaml 初始化配置
func (c *Config) InitFromEmbed() {
	data, err := configFile.ReadFile("config.yaml")
	if err != nil {
		log.Printf("[Config] default embedded file read error: %s", err)
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err = yaml.Unmarshal(data, c); err != nil {
		log.Printf("[Config] default embedded file parse error: %s", err)
	}
}

// Merge 合并另一个配置对象
func (c *Config) Merge(src *Config) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if src == nil {
		return
	}
	// 使用 mergo 进行深合并
	if err := mergo.Merge(c, src, mergo.WithOverride); err != nil {
		log.Printf("[Config] merge config error: %s", err)
	}
}

// View 读锁保护下的配置访问，自动处理加锁解锁
func (c *Config) View(f func(cfg *Config)) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	f(c)
}

// Update 写锁保护下的配置修改，自动处理加锁解锁
func (c *Config) Update(f func(cfg *Config)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	f(c)
}

// LicenseDaysLeft 计算 License 剩余天数
func (c *Config) LicenseDaysLeft() int64 {
	var expire string
	c.View(func(cfg *Config) {
		expire = strings.TrimSpace(cfg.NE.ExpiryDate)
	})

	if expire == "" || expire == "2000-00-00" || expire == "2000-01-01" {
		return -1
	}
	expireTime, err := time.Parse("2006-01-02", expire)
	if err != nil {
		return -1
	}
	daysLeft := time.Until(expireTime).Hours() / 24
	return int64(math.Ceil(daysLeft))
}

// RunTime 获取启动时间
func (c *Config) RunTime() time.Time {
	return c.startTime
}

// LoadExternalConfig 加载外部配置文件 支持 yaml/yml/json 格式
func LoadExternalConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	ext := strings.ToLower(filepath.Ext(configPath))
	switch ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, cfg)
	case ".json":
		err = json.Unmarshal(data, cfg)
	default:
		return nil, fmt.Errorf("unsupported config type: %s", ext)
	}

	if err != nil {
		return nil, err
	}
	return cfg, nil
}
