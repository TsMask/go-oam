package file

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriterFileTXTLine 逐行写入txt文件 sep 分割符号 需要转换数据
//
// 例如：
// data := [][]string{}
// data = append(data, []string{"姓名", "年龄", "城市"})
// data = append(data, []string{"1", "2", "3"})
// err := file.WriterFileTXT(data, filePath)
func WriterFileTXTLine(data [][]string, sep string, filePath string) error {
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
		line := strings.Join(row, sep)
		fmt.Fprintln(writer, line)
	}

	// 将缓冲区中的数据刷新到文件中
	err = writer.Flush()
	if err != nil {
		return err
	}
	return nil
}

// ReadFileTXTLine 逐行读取Txt文件，sep 分割符号 转换数组数据
func ReadFileTXTLine(sep string, filePath string) [][]string {
	// 创建 map 存储数据
	arr := make([][]string, 0)

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
		fields := strings.Split(line, sep)
		arr = append(arr, fields)
	}

	return arr
}

// WriterFileTXT 写入txt文件
//
// 例如：
// err := file.WriterFileTXT("", filePath)
func WriterFileTXT(text string, filePath string) error {
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

	// 将缓冲区中的数据刷新到文件中
	_, err = file.WriteString(text)
	if err != nil {
		return err
	}
	return nil
}
