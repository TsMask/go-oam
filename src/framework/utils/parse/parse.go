package parse

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Number 解析数值型
func Number(value any) int64 {
	switch v := value.(type) {
	case string:
		if v == "" {
			return 0
		}
		num, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0
		}
		return num
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(v).Int()
	case uint, uint8, uint16, uint32, uint64:
		return int64(reflect.ValueOf(v).Uint())
	case float32, float64:
		return int64(reflect.ValueOf(v).Float())
	case bool:
		if v {
			return 1
		}
		return 0
	default:
		return 0
	}
}

// Boolean 解析布尔型
func Boolean(value any) bool {
	switch v := value.(type) {
	case string:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return false
		}
		return b
	case int, int8, int16, int32, int64:
		num := reflect.ValueOf(v).Int()
		return num != 0
	case uint, uint8, uint16, uint32, uint64:
		num := int64(reflect.ValueOf(v).Uint())
		return num != 0
	case float32, float64:
		num := reflect.ValueOf(v).Float()
		return num != 0
	case bool:
		return v
	default:
		return false
	}
}

// ConvertToCamelCase 字符串转换驼峰形式
//
// 字符串 dict/inline/data/:dictId 结果 DictInlineDataDictId
func ConvertToCamelCase(str string) string {
	if len(str) == 0 {
		return str
	}
	reg := regexp.MustCompile(`[-_:/]\w`)
	result := reg.ReplaceAllStringFunc(str, func(match string) string {
		return strings.ToUpper(string(match[1]))
	})

	words := strings.Fields(result)
	for i, word := range words {
		str := word[1:]
		str = strings.ReplaceAll(str, "/", "")
		words[i] = strings.ToUpper(word[:1]) + str
	}

	return strings.Join(words, "")
}

// Bit 比特位为单位 1023.00 B --> 1.00 KB
func Bit(bit float64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}
	for i := 0; i < len(units); i++ {
		if bit < 1024 || i == len(units)-1 {
			return fmt.Sprintf("%.2f %s", bit, units[i])
		}
		bit /= 1024
	}
	return ""
}

// SafeContent 内容值进行安全掩码
func SafeContent(value string) string {
	if len(value) < 3 {
		return strings.Repeat("*", len(value))
	} else if len(value) < 6 {
		return string(value[0]) + strings.Repeat("*", len(value)-1)
	} else if len(value) < 10 {
		return string(value[0]) + strings.Repeat("*", len(value)-2) + string(value[len(value)-1])
	} else if len(value) < 15 {
		return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
	} else {
		return value[:3] + strings.Repeat("*", len(value)-6) + value[len(value)-3:]
	}
}

// RemoveDuplicates 数组内字符串去重
func RemoveDuplicates(arr []string) []string {
	uniqueIDs := make(map[string]bool)
	uniqueIDSlice := make([]string, 0)

	for _, id := range arr {
		_, ok := uniqueIDs[id]
		if !ok && id != "" {
			uniqueIDs[id] = true
			uniqueIDSlice = append(uniqueIDSlice, id)
		}
	}

	return uniqueIDSlice
}

// RemoveDuplicatesToArray 数组内字符串分隔去重转为字符数组
func RemoveDuplicatesToArray(keyStr, sep string) []string {
	arr := make([]string, 0)
	if keyStr == "" {
		return arr
	}
	if strings.Contains(keyStr, sep) {
		// 处理字符转数组后去重
		strArr := strings.Split(keyStr, sep)
		uniqueKeys := make(map[string]bool)
		for _, str := range strArr {
			_, ok := uniqueKeys[str]
			if !ok && str != "" {
				uniqueKeys[str] = true
				arr = append(arr, str)
			}
		}
	} else {
		arr = append(arr, keyStr)
	}
	return arr
}

// ConvertIPMask 转换IP网络地址掩码 24->"255.255.255.0" 20->"255.255.240.0"
func ConvertIPMask(bits int64) string {
	if bits < 0 || bits > 32 {
		return "255.255.255.255"
	}

	// 构建一个32位的uint32类型掩码，指定前bits位为1，其余为0
	mask := uint32((1<<bits - 1) << (32 - bits))

	// 将掩码转换为四个八位分组
	groups := []string{
		fmt.Sprintf("%d", mask>>24),
		fmt.Sprintf("%d", (mask>>16)&255),
		fmt.Sprintf("%d", (mask>>8)&255),
		fmt.Sprintf("%d", mask&255),
	}

	// 将分组用点号连接起来形成掩码字符串
	return strings.Join(groups, ".")
}

// ConvertConfigToMap 将配置内容转换为Map结构数据
//
// configType 类型支持：txt json yaml yml
func ConvertConfigToMap(configType, content string) (map[string]any, error) {
	// 类型支持：viper.SupportedExts
	// config := viper.New()
	// config.SetConfigType(configType)
	// err := config.ReadConfig(bytes.NewBuffer([]byte(content)))
	// return config.AllSettings(), err

	var configMap map[string]interface{}
	var err error
	if configType == "" || configType == "txt" {
		configMap = map[string]interface{}{
			"txt": content,
		}
	}
	if configType == "yaml" || configType == "yml" {
		err = yaml.Unmarshal([]byte(content), &configMap)
	}
	if configType == "json" {
		err = json.Unmarshal([]byte(content), &configMap)
	}
	return configMap, err
}

// ConvertConfigToFile 将数据写入到指定文件内
//
// configType 类型支持：txt json yaml yml
func ConvertConfigToFile(configType, filePath string, data any) error {
	// viper.SupportedExts
	// config := viper.New()
	// config.SetConfigType(configType)
	// for key, value := range mapData {
	// 	config.Set(key, value)
	// }
	// return config.WriteConfigAs(filePath)

	var dataByte []byte
	var err error
	if configType == "" || configType == "txt" {
		dataByte = []byte(data.(string))
	}
	if configType == "yaml" || configType == "yml" {
		dataByte, err = yaml.Marshal(data)
	}
	if configType == "json" {
		dataByte, err = json.Marshal(data)
	}
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0775); err != nil {
		return err
	}
	return os.WriteFile(filePath, dataByte, 0644)
}
