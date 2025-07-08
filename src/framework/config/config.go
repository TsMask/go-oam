package config

import (
	"bytes"
	"embed"
	"log"
	"math"
	"os"
	"time"

	"github.com/spf13/viper"
)

//go:embed config.yaml
var configFile embed.FS

// 程序配置
var conf *viper.Viper

// 初始化程序配置
func InitConfig() {
	conf = viper.New()
	initViper()
	// 记录程序开始运行的时间点
	conf.Set("runTime", time.Now())
}

// RunTime 程序开始运行的时间
func RunTime() time.Time {
	if conf == nil {
		return time.Time{}
	}
	return conf.GetTime("runTime")
}

// Enable 是否开启OAM
func Enable() bool {
	return conf.GetBool("enable")
}

// Dev 运行模式
func Dev() bool {
	return conf.GetBool("dev")
}

// LicenseDaysLeft 网元License剩余天数，小于0是过期
func LicenseDaysLeft() int64 {
	expire := conf.GetString("ne.expiryDate")
	if expire == "" || expire == "<nil>" {
		return -1
	}
	// 解析过期时间
	expireTime, err := time.Parse("2006-01-02", expire)
	if err != nil {
		return -1
	}
	// 计算距离天数，到结束日期计算是0
	daysLeft := time.Until(expireTime).Hours() / 24
	return int64(math.Ceil(daysLeft))
}

// 配置文件读取
func initViper() {
	// 如果配置文件名中没有扩展名，则需要设置Type
	conf.SetConfigType("yaml")
	// 读取默认配置文件
	configDefaultByte, err := configFile.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("config default file read error: %s", err)
		return
	}
	if err = conf.ReadConfig(bytes.NewReader(configDefaultByte)); err != nil {
		log.Fatalf("config default file read error: %s", err)
		return
	}
}

// Get 获取配置信息
//
// Get("ne.version")
func Get(key string) any {
	if conf == nil {
		return ""
	}
	return conf.Get(key)
}

// Set 修改配置信息
//
// Set("ne.version")
func Set(key string, value any) {
	if conf == nil {
		return
	}
	conf.Set(key, value)
}

// 程序配置
var confExt *viper.Viper

// ReadExternalConfig 读取外部文件配置
func ReadExternalConfig(configPath string) {
	f, err := os.Open(configPath)
	if err != nil {
		log.Fatalf("config external file read error: %s", err)
		return
	}
	defer f.Close()
	// 合并外部配置
	if err = conf.MergeConfig(f); err != nil {
		log.Fatalf("config external file merge error: %s", err)
		return
	}
	// 初始化外部配置
	confExt = viper.New()
	confExt.SetConfigType("yaml")
	if err = confExt.ReadConfig(f); err != nil {
		log.Fatalf("config external file read error: %s", err)
		return
	}
}

// GetExt 获取外部配置信息
//
// GetExt("server.0.ipv4")
func GetExt(key string) any {
	return confExt.Get(key)
}

// SetExt 修改外部配置信息
//
// SetExt("server.0.ipv4")
func SetExt(key string, value any) {
	confExt.Set(key, value)
	Set(key, value)
}

// WriteExternalConfig 写入外部文件配置
func WriteExternalConfig() {
	confExt.SafeWriteConfig()
}
