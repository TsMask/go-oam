package config

import (
	"bytes"
	"embed"
	"log"
	"math"
	"time"

	"github.com/spf13/viper"
)

//go:embed config.yaml
var configFile embed.FS

// 程序配置
var conf *viper.Viper = viper.New()

// 初始化程序配置
func InitConfig() {
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
	if conf == nil {
		return false
	}
	return conf.GetBool("enable")
}

// Dev 运行模式
func Dev() bool {
	if conf == nil {
		return false
	}
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
