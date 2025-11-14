package config

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed config.yaml
var configFile embed.FS

var conf map[string]any
var runTime time.Time

// 初始化程序配置
func InitConfig() {
	initConfigFromEmbed()
	runTime = time.Now()
}

// RunTime 程序开始运行的时间
func RunTime() time.Time {
	return runTime
}

// LicenseDaysLeft 网元License剩余天数，小于0是过期
func LicenseDaysLeft() int64 {
	expire := strings.TrimSpace(fmtString(Get("ne.expiryDate")))
	if expire == "" || expire == "<nil>" || expire == "2000-00-00" {
		return -1
	}
	expireTime, err := time.Parse("2006-01-02", expire)
	if err != nil {
		return -1
	}
	daysLeft := time.Until(expireTime).Hours() / 24
	return int64(math.Ceil(daysLeft))
}

func initConfigFromEmbed() {
	data, err := configFile.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("config default file read error: %s", err)
		return
	}
	var raw map[string]any
	if err = yaml.Unmarshal(data, &raw); err != nil {
		log.Fatalf("config default file parse error: %s", err)
		return
	}
	conf = normalizeKeys(raw)
}

func normalizeKeys(v any) map[string]any {
	m, ok := v.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	out := make(map[string]any, len(m))
	for k, val := range m {
		lk := strings.ToLower(k)
		out[lk] = normalizeValue(val)
	}
	return out
}

func normalizeValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return normalizeKeys(t)
	case []any:
		arr := make([]any, len(t))
		for i, e := range t {
			arr[i] = normalizeValue(e)
		}
		return arr
	default:
		return v
	}
}

// Get 获取配置信息
// Get("ne.version")
func Get(key string) any {
	if conf == nil {
		return ""
	}
	parts := strings.Split(key, ".")
	cur := any(conf)
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		m, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		val, exists := m[p]
		if !exists {
			// 尝试不区分大小写匹配
			found := false
			for mk, mv := range m {
				if strings.EqualFold(mk, p) {
					val = mv
					found = true
					break
				}
			}
			if !found {
				return ""
			}
		}
		cur = val
	}
	if cur == nil {
		return ""
	}
	return cur
}

// Set 修改配置信息
// Set("ne.version")
func Set(key string, value any) {
	if conf == nil {
		conf = map[string]any{}
	}
	parts := strings.Split(key, ".")
	cur := conf
	for i, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if i == len(parts)-1 {
			cur[p] = value
			return
		}
		next, ok := cur[p].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[p] = next
		}
		cur = next
	}
}

func fmtString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
