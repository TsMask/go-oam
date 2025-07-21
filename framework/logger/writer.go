package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// 日志器对象
type Logger struct {
	env         string         // 运行环境
	filePath    string         // 文件路径
	fileName    string         // 文件名
	level       int            // 日志等级标识
	maxDay      int            // 保留最长天数
	maxSize     int64          // 文件最大空间
	fileHandle  *os.File       // 文件实例
	logger      *log.Logger    // 日志实例
	logLevelMap map[int]string // 日志等级标识名
	logDay      int            // 日志当前日
}

const (
	LOG_LEVEL_SILENT = iota
	LOG_LEVEL_INFO
	LOG_LEVEL_WARN
	LOG_LEVEL_ERROR
)

// NewLogger 实例日志器对象
func NewLogger(dev bool, fileDir, fileName string, level, maxDay, maxSize int) (*Logger, error) {
	logFilePath := filepath.Join(fileDir, fileName)
	if err := os.MkdirAll(filepath.Dir(logFilePath), 0775); err != nil {
		return nil, fmt.Errorf("failed to mkdir logger dir: %v", err)
	}
	fileHandle, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	writer := io.Writer(fileHandle)
	if dev {
		writer = io.MultiWriter(fileHandle, os.Stderr)
	}

	logger := log.New(writer, "", log.LstdFlags|log.Lshortfile)

	logLevelMap := map[int]string{
		LOG_LEVEL_INFO:  "INFO",
		LOG_LEVEL_WARN:  "WARN",
		LOG_LEVEL_ERROR: "ERROR",
	}

	stdLogger := &Logger{
		filePath:    fileDir,
		fileName:    fileName,
		level:       level,
		maxDay:      maxDay,
		maxSize:     int64(maxSize * 1024 * 1024),
		fileHandle:  fileHandle,
		logger:      logger,
		logLevelMap: logLevelMap,
		logDay:      time.Now().Day(),
	}

	go stdLogger.checkFile()

	return stdLogger, nil
}

// checkFile 检查文件分割，自定时调用
func (l *Logger) checkFile() {
	fileInfo, err := l.fileHandle.Stat()
	if err != nil {
		l.logger.Printf("Failed to get log file info: %v\n", err)
		return
	}

	currTime := time.Now()
	if l.logDay != currTime.Day() {
		l.logDay = currTime.Day()
		l.rotateFile(currTime.AddDate(0, 0, -1).Format("2006-01-02"))
		// 移除超过保存最长天数的文件
		l.removeOldFile(currTime.AddDate(0, 0, -l.maxDay))
	} else if fileInfo.Size() >= l.maxSize {
		l.rotateFile(currTime.Format("2006-01-02_150405"))
	} else if time.Since(fileInfo.ModTime()).Hours() > 24 {
		l.rotateFile(fileInfo.ModTime().Format("2006-01-02"))
	}

	time.AfterFunc(1*time.Minute, l.checkFile)
}

// rotateFile 检查文件大小进行分割
func (l *Logger) rotateFile(timeFormat string) {
	l.fileHandle.Close()

	newFileName := fmt.Sprintf("%s.%s", l.fileName, timeFormat)
	newFilePath := filepath.Join(l.filePath, newFileName)
	oldfilePath := filepath.Join(l.filePath, l.fileName)

	// 重命名
	os.Rename(oldfilePath, newFilePath)

	// 新文件句柄
	fileHandle, err := os.OpenFile(oldfilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		l.logger.Printf("Failed to open log file: %v\n", err)
		return
	}

	l.fileHandle = fileHandle

	// 重新设置 logger 的 writer
	writer := io.Writer(l.fileHandle)
	if l.env == "local" {
		writer = io.MultiWriter(l.fileHandle, os.Stderr)
	}
	l.logger.SetOutput(writer)
}

// RemoveOldFile 删除旧文件
func (l *Logger) removeOldFile(oldFileDate time.Time) {
	// 遍历目标文件夹中的文件
	files, err := os.ReadDir(l.filePath)
	if err != nil {
		l.Errorf("logger RemoveOldFile ReadDir err: %v", err.Error())
		return
	}

	for _, file := range files {
		// 跳过非指定日志文件名
		if !strings.HasPrefix(file.Name(), l.fileName+".") {
			continue
		}
		idx := strings.LastIndex(file.Name(), ".")
		if idx == -1 {
			continue
		}
		dateStr := file.Name()[idx+1 : idx+11]

		// 解析日期字符串
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			l.Errorf("logger RemoveOldFile Parse err: %v", err.Error())
			continue
		}

		// 判断文件日期是否在给定日期之前
		if fileDate.Before(oldFileDate) {
			// 删除旧文件
			err := os.Remove(filepath.Join(l.filePath, file.Name()))
			if err != nil {
				l.Errorf("logger RemoveOldFile Remove err: %v", err.Error())
				continue
			}
		}
	}
}

// writeLog 写入chan
func (l *Logger) writeLog(level int, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	logMsg := fmt.Sprintf("[%s] %s\n", l.logLevelMap[level], fmt.Sprintf(format, args...))
	l.logger.Output(4, logMsg)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.writeLog(LOG_LEVEL_INFO, format, args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.writeLog(LOG_LEVEL_WARN, format, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.writeLog(LOG_LEVEL_ERROR, format, args...)
}

// Close 日志关闭
func (l *Logger) Close() {
	err := l.fileHandle.Close()
	if err != nil {
		l.logger.Printf("Failed to close log file: %v\n", err)
	}
}
