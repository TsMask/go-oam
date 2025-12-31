package file

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriterFileJSON 写入JSON文件
func WriterFileJSON(data any, filePath string) error {
	// 获取文件所在的目录路径
	dirPath := filepath.Dir(filePath)

	// 确保文件夹路径存在
	err := os.MkdirAll(dirPath, 0775)
	if err != nil {
		return err
	}

	// 创建或打开文件
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 创建 JSON 编码器
	encoder := json.NewEncoder(file)

	// 将数据编码并写入文件
	err = encoder.Encode(data)
	if err != nil {
		return err
	}

	return nil
}

// WriterFileJSONLine 写入JSON文件用 一行一个JSON
func WriterFileJSONLine(data []any, filePath string) error {
	// 获取文件所在的目录路径
	dirPath := filepath.Dir(filePath)

	// 确保文件夹路径存在
	err := os.MkdirAll(dirPath, 0775)
	if err != nil {
		return err
	}

	// 创建或打开文件
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 创建一个 Writer 对象，用于将数据写入文件
	writer := bufio.NewWriter(file)
	for _, row := range data {
		jsonData, errMarshal := json.Marshal(row)
		if errMarshal != nil {
			return errMarshal
		}

		// 写入 JSON 字符串到文件，并换行
		fmt.Fprintln(writer, string(jsonData))
	}

	// 将缓冲区中的数据刷新到文件中
	err = writer.Flush()
	if err != nil {
		return err
	}
	return nil
}

// ReadFileJSONLine 读取行JSON文件 一行一个JSON
func ReadFileJSONLine(filePath string) []string {
	// 创建 map 存储数据
	arr := make([]string, 0)

	// 打开文本文件
	file, err := os.Open(filePath)
	if err != nil {
		return arr
	}
	defer file.Close()

	// 创建一个 Scanner 对象，用于逐行读取文件内容
	scanner := bufio.NewScanner(file)
	if scanner.Err() != nil {
		return arr
	}

	for scanner.Scan() {
		line := scanner.Text()
		arr = append(arr, line)
	}

	return arr
}
