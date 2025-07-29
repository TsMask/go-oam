package config

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

// 程序配置
var confExt *viper.Viper = viper.New()

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
	confExt.SetConfigType("yaml")
	if err = confExt.ReadConfig(f); err != nil {
		log.Fatalf("config external file read error: %s", err)
		return
	}
}

// GetExt 获取外部配置信息
//
// GetExt("oamConfig.enable")
func GetExt(key string) any {
	return confExt.Get(key)
}

// SetExt 修改外部配置信息
//
// SetExt("oamConfig.enable", false)
func SetExt(key string, value any) {
	confExt.Set(key, value)
	Set(key, value)
}

// WriteExternalConfig 写入外部文件配置
func WriteExternalConfig() {
	confExt.SafeWriteConfig()
}
