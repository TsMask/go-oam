package file

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
)

// 写入CSV文件，需要转换数据
// 例如：
// data := [][]string{}
// data = append(data, []string{"姓名", "年龄", "城市"})
// data = append(data, []string{"1", "2", "3"})
// err := file.WriterFileCSV(data, filePath)
func WriterFileCSV(data [][]string, filePath string) error {
	// 创建本地输出目录
	if err := os.MkdirAll(filepath.Dir(filePath), 0775); err != nil {
		return err
	}

	// 创建或打开文件
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 创建CSV编写器
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入数据
	for _, row := range data {
		writer.Write(row)
	}
	return nil
}

// 读取CSV文件，转换map数据
func ReadFileCSV(filePath string) []map[string]string {
	// 创建 map 存储 CSV 数据
	arr := make([]map[string]string, 0)

	// 打开 CSV 文件
	file, err := os.Open(filePath)
	if err != nil {
		return arr
	}
	defer file.Close()

	// 创建 CSV Reader
	reader := csv.NewReader(file)

	// 读取 CSV 头部行
	header, err := reader.Read()
	if err != nil {
		return arr
	}

	// 遍历 CSV 数据行
	for {
		// 读取一行数据
		record, err := reader.Read()
		if err != nil {
			// 到达文件末尾或遇到错误时退出循环
			break
		}

		// 将 CSV 数据插入到 map 中
		data := make(map[string]string)
		for i, value := range record {
			key := strings.ToLower(header[i])
			data[key] = value
		}
		arr = append(arr, data)
	}

	return arr
}
