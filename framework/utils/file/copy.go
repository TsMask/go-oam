package file

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyFile 复制文件从 localPath 到 newPath
func CopyFile(localPath, newPath string) error {
	// 打开源文件
	srcFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer srcFile.Close()

	// 如果目标目录不存在，创建它
	if err := os.MkdirAll(filepath.Dir(newPath), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create destination directory: %v", err)
	}

	// 创建目标文件
	dstFile, err := os.Create(newPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer dstFile.Close()

	// 复制内容
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}

	// 返回成功
	return nil
}

// CopyDir 复制目录从 localDir 到 newDir
func CopyDir(localDir, newDir string) error {
	// 获取源目录中的所有文件和子目录
	entries, err := os.ReadDir(localDir)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %v", err)
	}

	// 如果目标目录不存在，创建它
	if err := os.MkdirAll(newDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create destination directory: %v", err)
	}

	// 遍历源目录中的每一个文件或子目录
	for _, entry := range entries {
		srcPath := filepath.Join(localDir, entry.Name())
		dstPath := filepath.Join(newDir, entry.Name())

		if entry.IsDir() {
			// 如果是目录，递归调用 CopyDir 复制子目录
			if err := CopyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// 如果是文件，调用 CopyFile 复制文件
			if err := CopyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
