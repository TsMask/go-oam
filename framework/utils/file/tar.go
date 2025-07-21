package file

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
)

// CompressTarGZByDir 将目录下文件添加到 tar.gz 压缩文件
func CompressTarGZByDir(zipFilePath, dirPath string) error {
	// 创建本地输出目录
	if err := os.MkdirAll(filepath.Dir(zipFilePath), 0775); err != nil {
		return err
	}

	// 创建输出文件
	tarFile, err := os.Create(zipFilePath)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	gw := gzip.NewWriter(tarFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

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
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}
		header.Name = relPath
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		// 打开文件
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// 写入文件内容
		_, err = io.Copy(tw, file)
		if err != nil {
			return err
		}

		return nil
	})

	return err
}
