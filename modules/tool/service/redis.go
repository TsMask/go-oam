package service

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/tsmask/go-oam/framework/route/resp"
	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/framework/ws/protocol"
	"github.com/tsmask/go-oam/modules/callback"
)

func NewRedisService() *Redis {
	return &Redis{}
}

// Redis 终端命令交互工具 服务层处理
type Redis struct {
}

// Command 执行单次命令 "GET key"
func (s *Redis) Command(handler callback.CallbackHandler, cmd string) (any, error) {
	if handler == nil {
		return "", fmt.Errorf("callback unrealized")
	}
	rdb := handler.Redis()
	if rdb == nil {
		return "", fmt.Errorf("redis client not connected")
	}

	client, ok := rdb.(*redis.Client)
	if !ok {
		return "", fmt.Errorf("redis client type error")
	}

	// 写入命令
	cmdArr := strings.Fields(cmd)
	if len(cmdArr) == 0 {
		return "", fmt.Errorf("redis command is empty")
	}

	args := make([]any, 0)
	for _, v := range cmdArr {
		args = append(args, v)
	}
	return client.Do(context.Background(), args...).Result()
}

// Session 接收终端交互业务处理
func (s Redis) Session(conn *ws.ServerConn, messageType int, req *protocol.Request) {
	switch req.Type {
	case "redis":
		// Redis会话消息接收写入会话
		command := fmt.Sprint(req.Data)
		if command == "" {
			conn.SendRespJSON(messageType, req.Uuid, resp.CODE_ERROR, "redis command is empty", nil)
			return
		}
		handler := conn.GetAnyConn().(callback.CallbackHandler)
		if handler == nil {
			conn.SendRespJSON(messageType, req.Uuid, resp.CODE_ERROR, "callback unrealized", nil)
			return
		}
		output, outerr := s.Command(handler, command)
		dataStr := ""
		if outerr != nil {
			dataStr = fmt.Sprintf("%s \r\n", outerr.Error())
		} else {
			// 获取结果的反射类型
			resultType := reflect.TypeOf(output)
			switch resultType.Kind() {
			case reflect.Slice:
				// 如果是切片类型需要进一步判断是否是 []string 或 []interface{}
				if resultType.Elem().Kind() == reflect.String {
					dataStr = fmt.Sprintf("%s \r\n", strings.Join(output.([]string), "\r\n"))
				} else if resultType.Elem().Kind() == reflect.Interface {
					arr := []string{}
					for _, v := range output.([]any) {
						arr = append(arr, fmt.Sprintf("%s", v))
					}
					dataStr = fmt.Sprintf("%s \r\n", strings.Join(arr, "\r\n"))
				}
			case reflect.Ptr:
				dataStr = "\r\n"
			case reflect.String, reflect.Int64:
				dataStr = fmt.Sprintf("%s \r\n", output)
			default:
				dataStr = fmt.Sprintf("%s \r\n", output)
			}
		}
		conn.SendRespJSON(messageType, req.Uuid, resp.CODE_SUCCESS, dataStr, nil)
	default:
		conn.SendRespJSON(messageType, req.Uuid, resp.CODE_ERROR, fmt.Sprintf("message type %s not supported", req.Type), nil)
		return
	}

}
