package service

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/tsmask/go-oam/framework/ws"
	"github.com/tsmask/go-oam/modules/callback"
	wsModel "github.com/tsmask/go-oam/modules/ws/model"
	wsService "github.com/tsmask/go-oam/modules/ws/service"
)

// 实例化服务层 Redis 结构体
var NewRedis = &Redis{}

// Redis 终端命令交互工具 服务层处理
type Redis struct{}

// Command 执行单次命令 "GET key"
func (s Redis) Command(cmd string) (any, error) {
	conn, err := callback.Redis()
	if err != nil {
		return "", err
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
	return conn.Do(context.Background(), args...).Result()
}

// Redis 接收终端交互业务处理
func (s Redis) Session(conn *ws.ServerConn, msg []byte) {
	var reqMsg wsModel.WSRequest
	if err := json.Unmarshal(msg, &reqMsg); err != nil {
		wsService.SendErr(conn, "", "message format json error")
		return
	}

	// 必传requestId确认消息
	if reqMsg.RequestID == "" {
		wsService.SendErr(conn, "", "message requestId is required")
		return
	}

	switch reqMsg.Type {
	case "close":
		conn.Close()
		return
	case "ping", "PING":
		conn.Pong()
		wsService.SendOK(conn, reqMsg.RequestID, "PONG")
		return
	case "redis":
		// Redis会话消息接收写入会话
		command := fmt.Sprint(reqMsg.Data)
		if command == "" {
			wsService.SendErr(conn, reqMsg.RequestID, "redis command is empty")
			return
		}
		output, outerr := s.Command(command)
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
		wsService.SendOK(conn, reqMsg.RequestID, dataStr)
	default:
		wsService.SendErr(conn, reqMsg.RequestID, fmt.Sprintf("message type %s not supported", reqMsg.Type))
		return
	}

}
