package config

import (
	"os"

	"github.com/spf13/viper"
)

// ReadExternalConfig 读取外部文件配置
// configPath 文件路径 /xx/xx/xx.yaml
// configType 文件类型 yaml json
func ReadExternalConfig(configPath, configType string) (*viper.Viper, error) {
	// 打开文件
	f, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// 初始化外部配置
	cfg := viper.New()
	cfg.SetConfigType(configType)
	if err = cfg.ReadConfig(f); err != nil {
		return nil, err
	}
	return cfg, nil
}
