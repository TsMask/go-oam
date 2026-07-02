package file

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyFile 复制文件，自动创建目标目录。
// 使用 32KB 缓冲区进行流式拷贝，适合大文件。
func CopyFile(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer src.Close()

	// 保留源文件权限，避免可执行文件复制后丢失执行位
	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}

	if err = os.MkdirAll(filepath.Dir(dstPath), 0775); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return fmt.Errorf("create dest: %w", err)
	}
	defer dst.Close()

	buf := make([]byte, 32*1024)
	_, err = io.CopyBuffer(dst, src, buf)
	if err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return nil
}

// CopyDir 递归复制目录。
func CopyDir(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("read dir: %w", err)
	}

	if err := os.MkdirAll(dstDir, 0775); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())

		if entry.IsDir() {
			if err := CopyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := CopyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}
