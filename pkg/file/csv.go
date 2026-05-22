package file

import (
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var csvMu sync.Mutex

// CSVWrite 写入 CSV 文件，线程安全。
// appendMode 为 true 时追加到已有文件，为 false 时覆盖写入（先写临时文件再重命名，保证原子性）。
func CSVWrite(filePath string, data [][]string, appendMode bool) error {
	csvMu.Lock()
	defer csvMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	var f *os.File
	var err error

	if appendMode {
		f, err = os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	} else {
		f, err = openTmp(filePath)
	}
	if err != nil {
		return err
	}
	defer func() {
		if !appendMode {
			f.Close()
			os.Remove(filePath + ".tmp")
		}
	}()

	writer := csv.NewWriter(f)
	for _, row := range data {
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if !appendMode {
		return os.Rename(filePath+".tmp", filePath)
	}
	return nil
}

// CSVRead 流式读取 CSV 文件，适用于大数据量场景。
// 每读取一行调用 fn 回调，fn 返回非 nil 错误时立即停止读取。
// header 传 nil 时自动读取首行作为表头，传非 nil 时跳过首行。
func CSVRead(filePath string, header []string, fn func(row map[string]string) error) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	if header == nil {
		header, err = reader.Read()
		if err != nil {
			return err
		}
		for i, h := range header {
			header[i] = strings.ToLower(strings.TrimSpace(h))
		}
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		row := make(map[string]string, len(header))
		for i, value := range record {
			if i < len(header) {
				row[header[i]] = value
			}
		}
		if err := fn(row); err != nil {
			return err
		}
	}
}

// CSVReadAll 读取整个 CSV 文件返回全部行，适用于小文件。
func CSVReadAll(filePath string) ([]map[string]string, error) {
	result := make([]map[string]string, 0, 64)
	err := CSVRead(filePath, nil, func(row map[string]string) error {
		result = append(result, row)
		return nil
	})
	return result, err
}
