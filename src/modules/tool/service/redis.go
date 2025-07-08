package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/tsmask/go-oam/src/callback"
	"github.com/tsmask/go-oam/src/framework/logger"
	"github.com/tsmask/go-oam/src/framework/route/resp"
	wsModel "github.com/tsmask/go-oam/src/modules/ws/model"
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
func (s Redis) Session(client *wsModel.WSClient, reqMsg wsModel.WSRequest) {
	// 必传requestId确认消息
	if reqMsg.RequestID == "" {
		msg := "message requestId is required"
		logger.Infof("ws Redis UID %s err: %s", client.BindUid, msg)
		msgByte, _ := json.Marshal(resp.ErrMsg(msg))
		client.MsgChan <- msgByte
		return
	}

	var resByte []byte
	var err error

	switch reqMsg.Type {
	case "close":
		// 主动关闭
		resultByte, _ := json.Marshal(resp.OkMsg("user initiated closure"))
		client.MsgChan <- resultByte
		// 等待1s后关闭连接
		time.Sleep(1 * time.Second)
		client.StopChan <- struct{}{}
		return
	case "redis":
		// Redis会话消息接收写入会话
		command := fmt.Sprint(reqMsg.Data)
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
		resByte, _ = json.Marshal(resp.Ok(map[string]any{
			"requestId": reqMsg.RequestID,
			"data":      dataStr,
		}))
	default:
		err = fmt.Errorf("message type %s not supported", reqMsg.Type)
	}

	if err != nil {
		logger.Warnf("ws Redis UID %s err: %s", client.BindUid, err.Error())
		msgByte, _ := json.Marshal(resp.ErrMsg(err.Error()))
		client.MsgChan <- msgByte
		if err == io.EOF {
			// 等待1s后关闭连接
			time.Sleep(1 * time.Second)
			client.StopChan <- struct{}{}
		}
		return
	}
	if len(resByte) > 0 {
		client.MsgChan <- resByte
	}
}
