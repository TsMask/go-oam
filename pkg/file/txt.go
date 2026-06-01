package file

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var txtMu sync.Mutex

// TXTWrite 写入纯文本文件（原子写入，线程安全）。
func TXTWrite(filePath string, text string) error {
	txtMu.Lock()
	defer txtMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(filePath+".tmp", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		os.Remove(filePath + ".tmp")
	}()

	if _, err := f.WriteString(text); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(filePath+".tmp", filePath)
}

// TXTRead 读取整个文本文件内容。
func TXTRead(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	return string(data), err
}

// TXTLineWrite 按行写入分隔文本文件（原子写入，线程安全）。
// sep 为列分隔符，每行用 sep 连接各字段后写入。
func TXTLineWrite(filePath string, sep string, data [][]string) error {
	txtMu.Lock()
	defer txtMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(filePath+".tmp", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		os.Remove(filePath + ".tmp")
	}()

	writer := bufio.NewWriter(f)
	for _, row := range data {
		if _, err := fmt.Fprintln(writer, strings.Join(row, sep)); err != nil {
			return err
		}
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(filePath+".tmp", filePath)
}

// TXTLineRead 流式读取分隔文本文件，适用于大数据量场景。
// sep 为列分隔符，每行按 sep 拆分为数组后回调 fn。
// fn 返回非 nil 错误时立即停止读取。
func TXTLineRead(filePath string, sep string, fn func(fields []string) error) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), sep)
		if err := fn(fields); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// TXTLineReadAll 读取整个分隔文本文件返回全部行，适用于小文件。
func TXTLineReadAll(filePath string, sep string) ([][]string, error) {
	result := make([][]string, 0, 64)
	err := TXTLineRead(filePath, sep, func(fields []string) error {
		result = append(result, fields)
		return nil
	})
	return result, err
}
