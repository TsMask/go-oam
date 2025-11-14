package file

import (
    "fmt"
    "io"
    "os"
    "mime/multipart"
    "path/filepath"
    "sort"
    "strconv"
)

// transferToNewFile 读取目标文件转移到新路径下
//
// readFilePath 读取目标文件
//
// writePath 写入路径
//
// fileName 文件名称
func transferToNewFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	if err = os.MkdirAll(filepath.Dir(dst), 0775); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}

func writeBytesToFile(dst string, data []byte) error {
    if err := os.MkdirAll(filepath.Dir(dst), 0775); err != nil {
        return err
    }
    out, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer out.Close()
    _, err = out.Write(data)
    return err
}

// mergeToNewFile 将多个文件合并成一个文件并删除合并前的切片目录文件
//
// dirPath 读取要合并文件的目录
//
// writePath 写入路径
func mergeToNewFile(dirPath string, writePath string) error {
	// 读取目录下所有文件并排序，注意文件名称是否数值
	fileNameList, err := getDirFileNameList(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read merge target file: %v", err)
	}
	if len(fileNameList) <= 0 {
		return fmt.Errorf("failed to read merge target file")
	}

	// 排序
	sort.Slice(fileNameList, func(i, j int) bool {
		numI, _ := strconv.Atoi(fileNameList[i])
		numJ, _ := strconv.Atoi(fileNameList[j])
		return numI < numJ
	})

	// 写入到新路径文件
	if err = os.MkdirAll(filepath.Dir(writePath), 0775); err != nil {
		return err
	}

	// 转移完成后删除切片目录
	defer os.Remove(dirPath)

	// 打开新路径文件
	outputFile, err := os.Create(writePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer outputFile.Close()

	// 逐个读取文件后进行流拷贝数据块
	for _, fileName := range fileNameList {
		chunkPath := filepath.Join(dirPath, fileName)
		// 拷贝结束后删除切片
		defer os.Remove(chunkPath)
		// 打开切片文件
		inputFile, err := os.Open(chunkPath)
		if err != nil {
			return fmt.Errorf("failed to open file: %v", err)
		}
		defer inputFile.Close()
		// 拷贝文件流
		_, err = io.Copy(outputFile, inputFile)
		if err != nil {
			return fmt.Errorf("failed to copy file data: %w", err)
		}
	}

	return nil
}

// getFileSize 读取文件大小
func getFileSize(filePath string) int64 {
	// 获取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return 0
	}
	// 获取文件大小（字节数）
	return fileInfo.Size()
}

// 读取文件流用于返回下载
//
// filePath 文件路径
// startOffset, endOffset 分片块读取区间，根据文件切片的块范围
func getFileStream(filePath string, startOffset, endOffset int64) ([]byte, error) {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 获取文件的大小
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := fileInfo.Size()

	// 确保起始和结束偏移量在文件范围内
	if startOffset > fileSize {
		startOffset = 0
	}
	if endOffset >= fileSize {
		endOffset = fileSize - 1
	}

	// 计算切片的大小
	chunkSize := endOffset - startOffset + 1

	// 创建 SectionReader
	reader := io.NewSectionReader(file, startOffset, chunkSize)

	// 创建一个缓冲区来存储读取的数据
	buffer := make([]byte, chunkSize)

	// 读取数据到缓冲区
	_, err = reader.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return buffer, nil
}

// 获取文件目录下所有文件名称，不含目录名称
//
// filePath 文件路径
func getDirFileNameList(dirPath string) ([]string, error) {
	fileNames := []string{}

	dir, err := os.Open(dirPath)
	if err != nil {
		return fileNames, nil
	}
	defer dir.Close()

	fileInfos, err := dir.Readdir(-1)
	if err != nil {
		return fileNames, err
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.Mode().IsRegular() {
			fileNames = append(fileNames, fileInfo.Name())
		}
	}

	return fileNames, nil
}
