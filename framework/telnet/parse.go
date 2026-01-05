package telnet

import (
	"strings"
)

// ConvertToStr 转换为string
func ConvertToStr(output string) (string, error) {
	index := strings.Index(output, "\n")
	if index != -1 {
		output = output[:index]
	}
	return strings.TrimSpace(output), nil
}

// ConvertToMap 转换为map
func ConvertToMap(output string) (map[string]string, error) {
	// 初始化一个map用于存储拆分后的键值对
	m := make(map[string]string)

	var items []string
	if strings.Contains(output, "\r\n") {
		// 按照分隔符"\r\n"进行拆分
		items = strings.Split(output, "\r\n")
	} else if strings.Contains(output, "\n") {
		// 按照分隔符"\n"进行拆分
		items = strings.Split(output, "\n")
	}

	// 遍历拆分后的结果
	for _, item := range items {
		var pair []string

		if strings.Contains(item, "=") {
			// 按照分隔符"="进行拆分键值对
			pair = strings.SplitN(item, "=", 2)
		} else if strings.Contains(item, ":") {
			// 按照分隔符":"进行拆分键值对
			pair = strings.SplitN(item, ":", 2)
		}

		if len(pair) == 2 {
			// 将键值对存入map中
			m[pair[0]] = pair[1]
		}
	}
	return m, nil
}
