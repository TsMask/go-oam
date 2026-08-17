package file

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

// zipBufSize zip 读写缓冲区大小 32KB
const zipBufSize = 32 * 1024

// ZipPackDir 将目录打包为 zip 文件。
func ZipPackDir(dirPath, zipPath string) error {
	return withPathLock(zipPath, func() error {
		if err := os.MkdirAll(filepath.Dir(zipPath), 0755); err != nil {
			return err
		}
		return writeFileAtomic(zipPath, 0644, func(f *os.File) error {
			return zipPackDirWriter(dirPath, f)
		})
	})
}

func zipPackDirWriter(dirPath string, f *os.File) error {
	zw := zip.NewWriter(f)

	walkErr := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
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
		// ZIP 规范要求条目名使用正斜杠，避免 Windows 反斜杠导致目录结构丢失
		header.Name = filepath.ToSlash(relPath)
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

	// 显式关闭以确保中央目录写入成功，错误不可忽略
	if cerr := zw.Close(); walkErr == nil {
		walkErr = cerr
	}
	return walkErr
}

// ZipPackFile 将单个文件打包为 zip 文件。
func ZipPackFile(filePath, zipPath string) error {
	return withPathLock(zipPath, func() error {
		if err := os.MkdirAll(filepath.Dir(zipPath), 0755); err != nil {
			return err
		}
		return writeFileAtomic(zipPath, 0644, func(out *os.File) error {
			return zipPackFileWriter(filePath, out)
		})
	})
}

func zipPackFileWriter(filePath string, out *os.File) error {
	zw := zip.NewWriter(out)

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
	// 显式关闭以确保中央目录写入成功，错误不可忽略
	if cerr := zw.Close(); err == nil {
		err = cerr
	}
	return err
}

// ZipUnpack 解压 zip 文件到目录。
func ZipUnpack(zipPath, dirPath string) error {
	return withPathLock(dirPath, func() error {
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
			target, targetErr := archiveTarget(dirPath, f.Name)
			if targetErr != nil {
				return targetErr
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

			dst, createErr := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode().Perm())
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
	})
}
