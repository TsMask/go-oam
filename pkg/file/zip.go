package file

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// zipBufSize zip 读写缓冲区大小 32KB
const zipBufSize = 32 * 1024

// ZipPackDir 将目录打包为 zip 文件。
func ZipPackDir(dirPath, zipPath string) error {
	if err := os.MkdirAll(filepath.Dir(zipPath), 0755); err != nil {
		return err
	}
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath
		header.Method = zip.Deflate

		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()

		buf := make([]byte, zipBufSize)
		_, err = io.CopyBuffer(writer, src, buf)
		return err
	})
}

// ZipPackFile 将单个文件打包为 zip 文件。
func ZipPackFile(filePath, zipPath string) error {
	if err := os.MkdirAll(filepath.Dir(zipPath), 0755); err != nil {
		return err
	}
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	src, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = info.Name()
	header.Method = zip.Deflate

	writer, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	buf := make([]byte, zipBufSize)
	_, err = io.CopyBuffer(writer, src, buf)
	return err
}

// ZipUnpack 解压 zip 文件到目录。
func ZipUnpack(zipPath, dirPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}

	buf := make([]byte, zipBufSize)
	for _, f := range r.File {
		// 防止路径穿越攻击
		target := filepath.Join(dirPath, f.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dirPath)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		dst, createErr := os.Create(target)
		if createErr != nil {
			rc.Close()
			return createErr
		}
		_, err = io.CopyBuffer(dst, rc, buf)
		dst.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
