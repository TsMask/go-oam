package config

import (
    "encoding/json"
    "os"
    "strings"

    "gopkg.in/yaml.v3"
)

// ReadExternalConfig 读取外部文件配置
// configPath 文件路径 /xx/xx/xx.yaml
// configType 文件类型 yaml json
func ReadExternalConfig(configPath, configType string) (map[string]any, error) {
    data, err := os.ReadFile(configPath)
    if err != nil {
        return nil, err
    }
    var raw map[string]any
    switch strings.ToLower(strings.TrimSpace(configType)) {
    case "yaml", "yml":
        if err = yaml.Unmarshal(data, &raw); err != nil {
            return nil, err
        }
    case "json":
        if err = json.Unmarshal(data, &raw); err != nil {
            return nil, err
        }
    default:
        if err = yaml.Unmarshal(data, &raw); err != nil {
            return nil, err
        }
    }
    return normalizeKeys(raw), nil
}
