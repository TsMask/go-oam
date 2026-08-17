package file

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
)

// JSONWrite 写入 JSON 文件（原子写入，线程安全）。
func JSONWrite(filePath string, data any) error {
	return withPathLock(filePath, func() error {
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}
		return writeFileAtomic(filePath, 0644, func(f *os.File) error {
			encoder := json.NewEncoder(f)
			encoder.SetIndent("", "  ")
			return encoder.Encode(data)
		})
	})
}

func JSONRead(filePath string, data any) error {
	return withPathLock(filePath, func() error {
		f, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer f.Close()
		return json.NewDecoder(f).Decode(data)
	})
}

// JSONLineWrite 写入 JSON Lines 文件，每行一个 JSON 对象（原子写入，线程安全）。
func JSONLineWrite(filePath string, data []any) error {
	return withPathLock(filePath, func() error {
		return jsonLineWriteLocked(filePath, data)
	})
}

func jsonLineWriteLocked(filePath string, data []any) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	return writeFileAtomic(filePath, 0644, func(f *os.File) error {
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
		return writer.Flush()
	})
}

// JSONLineAppend 追加一行 JSON 到文件（线程安全）。
func JSONLineAppend(filePath string, data any) error {
	return withPathLock(filePath, func() error {
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
		if _, err := f.Write(line); err != nil {
			return err
		}
		return f.Sync()
	})
}

// JSONLineRead 流式读取 JSON Lines 文件，适用于大数据量场景。
// 每读取一行调用 fn 回调，fn 返回非 nil 错误时立即停止读取。
func JSONLineRead(filePath string, fn func(line string) error) error {
	return withPathLock(filePath, func() error {
		return jsonLineReadLocked(filePath, fn)
	})
}

func jsonLineReadLocked(filePath string, fn func(line string) error) error {
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
