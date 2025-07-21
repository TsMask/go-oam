package file

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// UnZip 解 ZIP 压缩文件输出到目录下
func UnZip(zipFilePath, dirPath string) error {
	// 打开ZIP文件进行读取
	r, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return err
	}
	defer r.Close()

	// 创建本地输出目录
	if err := os.MkdirAll(dirPath, 0775); err != nil {
		return err
	}

	// 遍历ZIP文件中的每个文件并解压缩到输出目录
	for _, f := range r.File {
		// 打开ZIP文件中的文件
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		// 创建解压后的文件
		path := filepath.ToSlash(filepath.Join(dirPath, f.Name))
		if f.FileInfo().IsDir() {
			// 如果是目录，创建目录
			if err := os.MkdirAll(path, 0775); err != nil {
				return err
			}
		} else {
			if err = os.MkdirAll(filepath.Dir(path), 0775); err != nil {
				return err
			}
			out, err := os.Create(path)
			if err != nil {
				return err
			}
			defer out.Close()

			if _, err = io.Copy(out, rc); err != nil {
				return err
			}
		}
	}

	return nil
}

// CompressZipByFile 将单文件添加到 ZIP 压缩文件
func CompressZipByFile(zipFilePath, filePath string) error {
	// 创建一个新的 ZIP 文件
	newZipFile, err := os.Create(zipFilePath)
	if err != nil {
		return fmt.Errorf("创建 ZIP 文件失败: %w", err)
	}
	defer newZipFile.Close()

	// 创建 ZIP 写入器
	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	fileToCompress, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer fileToCompress.Close()

	// 获取文件信息
	fileInfo, err := fileToCompress.Stat()
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 创建文件头
	header, err := zip.FileInfoHeader(fileInfo)
	if err != nil {
		return fmt.Errorf("创建文件头失败: %w", err)
	}

	// 设置文件头中的名称
	header.Name = fileInfo.Name()

	// 创建文件在 ZIP 中的写入器
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("创建文件写入器失败: %w", err)
	}

	// 将文件内容复制到 ZIP 文件中
	_, err = io.Copy(writer, fileToCompress)
	if err != nil {
		return fmt.Errorf("将文件内容复制到 ZIP 失败: %w", err)
	}

	return nil
}

// CompressZipByDir 将目录下文件添加到 ZIP 压缩文件
func CompressZipByDir(zipFilePath, dirPath string) error {
	// 创建本地输出目录
	if err := os.MkdirAll(filepath.Dir(zipFilePath), 0775); err != nil {
		return err
	}

	// 创建输出文件
	zipWriter, err := os.Create(zipFilePath)
	if err != nil {
		return err
	}
	defer zipWriter.Close()

	// 创建 zip.Writer
	zipWriterObj := zip.NewWriter(zipWriter)
	defer zipWriterObj.Close()

	// 遍历目录下的所有文件和子目录
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 忽略目录
		if info.IsDir() {
			return nil
		}

		// 创建文件条目
		relativePath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}
		fileEntry, err := zipWriterObj.Create(relativePath)
		if err != nil {
			return err
		}

		// 打开文件
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// 写入文件内容到 ZIP 文件
		_, err = io.Copy(fileEntry, file)
		if err != nil {
			return err
		}

		return nil
	})

	return err
}
