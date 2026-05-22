package file

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// tarBufSize tar 读写缓冲区大小 32KB
const tarBufSize = 32 * 1024

// TarPack 将目录打包为普通 tar 文件。
func TarPack(dirPath, tarPath string) error {
	f, err := os.Create(tarPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return tarPackWalk(dirPath, tar.NewWriter(f))
}

// TarUnpack 解包普通 tar 文件到目录。
func TarUnpack(tarPath, dirPath string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return tarUnpackReader(f, dirPath)
}

// TarGzPack 将目录打包为 tar.gz 文件。
func TarGzPack(dirPath, tarGzPath string) error {
	if err := os.MkdirAll(filepath.Dir(tarGzPath), 0755); err != nil {
		return err
	}
	f, err := os.Create(tarGzPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gw, err := gzip.NewWriterLevel(f, gzip.BestSpeed)
	if err != nil {
		return err
	}
	defer gw.Close()

	return tarPackWalk(dirPath, tar.NewWriter(gw))
}

// TarGzUnpack 解压 tar.gz 文件到目录。
func TarGzUnpack(tarGzPath, dirPath string) error {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	return tarUnpackReader(gr, dirPath)
}

// tarPackWalk 遍历目录并将所有文件写入 tar writer。
func tarPackWalk(dirPath string, tw *tar.Writer) error {
	defer tw.Close()

	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}

		// 写入文件头
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// 目录和非常规文件只写 header
		if !info.Mode().IsRegular() {
			return nil
		}

		// 写入文件内容，使用预分配缓冲区
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()

		buf := make([]byte, tarBufSize)
		_, err = io.CopyBuffer(tw, src, buf)
		return err
	})
}

// tarUnpackReader 从 tar reader 流式解包到目录。
func tarUnpackReader(r io.Reader, dirPath string) error {
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}

	tr := tar.NewReader(r)
	buf := make([]byte, tarBufSize)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// 防止路径穿越攻击
		target := filepath.Join(dirPath, header.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dirPath)+string(os.PathSeparator)) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			if err := copyToFile(target, tr, buf); err != nil {
				return err
			}
		}
	}
}
