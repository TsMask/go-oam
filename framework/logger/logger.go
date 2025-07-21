package logger

import (
	"log"

	"github.com/tsmask/go-oam/framework/config"
)

var logWriter *Logger

// 初始程序日志
func InitLogger() {
	dev := config.Dev()
	conf := config.Get("logger").(map[string]any)
	fileDir := conf["filedir"].(string)
	fileName := conf["filename"].(string)
	level := conf["level"].(int)
	maxDay := conf["maxday"].(int)
	maxSize := conf["maxsize"].(int)

	newLog, err := NewLogger(dev, fileDir, fileName, level, maxDay, maxSize)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	logWriter = newLog
}

// 关闭程序日志写入
func Close() {
	if logWriter != nil {
		logWriter.Close()
	}
}

func Infof(format string, v ...any) {
	logWriter.Infof(format, v...)
}

func Warnf(format string, v ...any) {
	logWriter.Warnf(format, v...)
}

func Errorf(format string, v ...any) {
	logWriter.Errorf(format, v...)
}

// Fatalf 抛出错误并退出程序
func Fatalf(format string, v ...any) {
	log.Fatalf(format, v...)
}
