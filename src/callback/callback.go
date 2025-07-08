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
	SNMP(command string) string
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
			return rdb.(*redis.Client), nil
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
func SNMP(command string) string {
	if invoke != nil {
		return invoke.Telent(command)
	}
	return "snmp unrealized"
}
