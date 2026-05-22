package file

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

var jsonMu sync.Mutex

// JSONWrite 写入 JSON 文件（原子写入，线程安全）。
func JSONWrite(filePath string, data any) error {
	jsonMu.Lock()
	defer jsonMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	f, err := openTmp(filePath)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		os.Remove(filePath + ".tmp")
	}()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(filePath+".tmp", filePath)
}

// JSONRead 读取 JSON 文件并解码到 data。
func JSONRead(filePath string, data any) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(data)
}

// JSONLineWrite 写入 JSON Lines 文件，每行一个 JSON 对象（原子写入，线程安全）。
func JSONLineWrite(filePath string, data []any) error {
	jsonMu.Lock()
	defer jsonMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	f, err := openTmp(filePath)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		os.Remove(filePath + ".tmp")
	}()

	writer := bufio.NewWriter(f)
	for _, row := range data {
		line, err := json.Marshal(row)
		if err != nil {
			return err
		}
		line = append(line, '\n')
		if _, err := writer.Write(line); err != nil {
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

// JSONLineAppend 追加一行 JSON 到文件（线程安全）。
func JSONLineAppend(filePath string, data any) error {
	jsonMu.Lock()
	defer jsonMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	line, err := json.Marshal(data)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	_, err = f.Write(line)
	return err
}

// JSONLineRead 流式读取 JSON Lines 文件，适用于大数据量场景。
// 每读取一行调用 fn 回调，fn 返回非 nil 错误时立即停止读取。
func JSONLineRead(filePath string, fn func(line string) error) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if err := fn(scanner.Text()); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// JSONLineReadAll 读取整个 JSON Lines 文件返回全部行，适用于小文件。
func JSONLineReadAll(filePath string) ([]string, error) {
	result := make([]string, 0, 64)
	err := JSONLineRead(filePath, func(line string) error {
		result = append(result, line)
		return nil
	})
	return result, err
}
