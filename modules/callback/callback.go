package callback

import (
	"fmt"

	"github.com/redis/go-redis/v9"
)

var invoke CallbackHandler

// Handler 回调实现
func Handler(callback CallbackHandler) {
	invoke = callback
}

// 外部回调接口
type CallbackHandler interface {
	// 备用状态
	Standby() bool
	// Redis 实例 *redis.Client
	Redis() any
	// Telent 消息处理
	Telent(command string) string
	// SNMP 消息处理
	SNMP(oid, operType string, value any) any
	// Config 网元配置消息处理
	// action  操作行为 C/U/R/D Create（创建）、Update（更新）、Read（读取）、Delete（删除）
	// paramName 参数名称
	// loc 定位符 list为空字符 array使用与数据对象内index一致,有多层时划分嵌套层(index/subParamName/index)
	// paramValue 参数值
	Config(action, paramName, loc string, paramValue any) error
}

// Standby 备用状态
func Standby() bool {
	if invoke != nil {
		return invoke.Standby()
	}
	return false
}

// Redis 实例 *redis.Client
func Redis() (*redis.Client, error) {
	err := fmt.Errorf("redis client not connected")
	if invoke != nil {
		rdb := invoke.Redis()
		if rdb != nil {
			client, ok := rdb.(*redis.Client)
			if ok {
				return client, nil
			}
		}
	}
	return nil, err
}

// Telent 消息处理
func Telent(command string) string {
	if invoke != nil {
		return invoke.Telent(command)
	}
	return "telent unrealized"
}

// SNMP 消息处理
func SNMP(oid, operType string, value any) any {
	if invoke != nil {
		return invoke.SNMP(oid, operType, value)
	}
	return "snmp unrealized"
}

// Config 网元配置消息处理
// action  操作行为 Create（创建）、Update（更新）、Read（读取）、Delete（删除）
// paramName 参数名称
// loc 定位符 list为空字符 array使用与数据对象内index一致,有多层时划分嵌套层(index/subParamName/index)
// paramValue 参数值
func Config(action, paramName, loc string, paramValue any) error {
	if invoke != nil {
		return invoke.Config(action, paramName, loc, paramValue)
	}
	return fmt.Errorf("config unrealized => %s > %s > %s > %v", action, paramName, loc, paramValue)
}
