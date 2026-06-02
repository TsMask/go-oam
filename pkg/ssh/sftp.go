package ssh

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	gosftp "github.com/pkg/sftp"
)

// === 类型定义

// FileEntry 文件信息
type FileEntry struct {
	FileName     string `json:"fileName"`     // 文件名
	FilePath     string `json:"filePath"`     // 完整路径
	FileType     string `json:"fileType"`     // 类型: dir, file, symlink
	FileMode     string `json:"fileMode"`     // 权限字符串如 "rwxr-xr-x"
	FileSize     int64  `json:"fileSize"`     // 文件大小(字节)
	ModifiedTime int64  `json:"modifiedTime"` // 最后修改时间(Unix毫秒)
}

// === SFTP

// SFTP 文件传输客户端
//
// 通过 Client.NewSFTP 创建，支持文件列表、上传下载、目录递归传输和过期文件清理。
// 使用完毕后须调用 Close 释放资源。
type SFTP struct {
	client    *gosftp.Client
	closeOnce sync.Once
}

// NewSFTP 创建SFTP文件传输客户端
func (c *Client) NewSFTP() (*SFTP, error) {
	sftpClient, err := gosftp.NewClient(c.sshClient)
	if err != nil {
		return nil, fmt.Errorf("ssh sftp: 创建SFTP客户端失败: %w", err)
	}
	return &SFTP{client: sftpClient}, nil
}

// Close 关闭SFTP客户端
func (s *SFTP) Close() {
	s.closeOnce.Do(func() {
		if s.client != nil {
			s.client.Close()
		}
	})
}

// Client 获取SFTP客户端
func (s *SFTP) Client() *gosftp.Client {
	if s.client != nil {
		return s.client
	}
	return nil
}

// === 文件列表

// ListDir 列出远程目录下的文件
//
// 通过 SFTP 协议 ReadDir 获取文件属性。pattern 为 glob 匹配模式（如 "*.log"），为空列出所有。
// 结果按修改时间倒序排列。
func (s *SFTP) ListDir(remoteDir, pattern string) ([]FileEntry, error) {
	var names []string
	if pattern != "" {
		// Glob 模糊匹配
		matches, err := s.client.Glob(filepath.Join(remoteDir, pattern))
		if err != nil {
			return nil, fmt.Errorf("ssh sftp: 匹配文件失败: %w", err)
		}
		names = matches
	}

	entries, err := s.client.ReadDir(remoteDir)
	if err != nil {
		return nil, fmt.Errorf("ssh sftp: 读取远程目录失败: %w", err)
	}

	// 构建 glob 匹配集合
	globSet := make(map[string]bool, len(names))
	for _, n := range names {
		globSet[filepath.Base(n)] = true
	}

	var files []FileEntry
	for _, entry := range entries {
		name := entry.Name()

		// 如果有 glob 条件且当前名称不匹配，跳过
		if pattern != "" && !globSet[name] {
			continue
		}

		files = append(files, newFileEntry(entry, name, remoteDir))
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ModifiedTime > files[j].ModifiedTime
	})
	return files, nil
}

// Stat 获取单个远程文件的详细信息
func (s *SFTP) Stat(remotePath string) (FileEntry, error) {
	info, err := s.client.Stat(remotePath)
	if err != nil {
		return FileEntry{}, fmt.Errorf("ssh sftp: 获取文件信息失败: %w", err)
	}
	return newFileEntry(info, filepath.Base(remotePath), filepath.Dir(remotePath)), nil
}

// Exists 检查远程文件或目录是否存在
func (s *SFTP) Exists(remotePath string) bool {
	_, err := s.client.Stat(remotePath)
	return err == nil
}

// newFileEntry 从 os.FileInfo 构造 FileEntry
func newFileEntry(info os.FileInfo, name, dirPath string) FileEntry {
	fileType := "file"
	if info.IsDir() {
		fileType = "dir"
	} else if info.Mode()&os.ModeSymlink != 0 {
		fileType = "symlink"
	}
	return FileEntry{
		FileName:     name,
		FilePath:     filepath.Join(dirPath, name),
		FileType:     fileType,
		FileMode:     info.Mode().String(),
		FileSize:     info.Size(),
		ModifiedTime: info.ModTime().UnixMilli(),
	}
}

// === 文件管理

// RemoveOldFiles 删除远程目录下修改时间小于 expireTime 的文件
func (s *SFTP) RemoveOldFiles(remoteDir string, expireTime time.Time) error {
	entries, err := s.client.ReadDir(remoteDir)
	if err != nil {
		return fmt.Errorf("ssh sftp: 读取远程目录失败: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if entry.ModTime().Before(expireTime) {
			filePath := filepath.ToSlash(filepath.Join(remoteDir, entry.Name()))
			if err := s.client.Remove(filePath); err != nil {
				continue
			}
		}
	}
	return nil
}

// === 单文件传输

// Upload 上传本地文件到远程
func (s *SFTP) Upload(localPath, remotePath string) error {
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("ssh sftp: 打开本地文件失败: %w", err)
	}
	defer localFile.Close()

	if err := s.client.MkdirAll(filepath.Dir(remotePath)); err != nil {
		return fmt.Errorf("ssh sftp: 创建远程目录失败: %w", err)
	}

	remoteFile, err := s.client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("ssh sftp: 创建远程文件失败: %w", err)
	}
	defer remoteFile.Close()

	if _, err := io.Copy(remoteFile, localFile); err != nil {
		return fmt.Errorf("ssh sftp: 上传文件失败: %w", err)
	}
	return nil
}

// Download 下载远程文件到本地
func (s *SFTP) Download(remotePath, localPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0775); err != nil {
		return fmt.Errorf("ssh sftp: 创建本地目录失败: %w", err)
	}

	remoteFile, err := s.client.Open(remotePath)
	if err != nil {
		return fmt.Errorf("ssh sftp: 打开远程文件失败: %w", err)
	}
	defer remoteFile.Close()

	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("ssh sftp: 创建本地文件失败: %w", err)
	}
	defer localFile.Close()

	if _, err := io.Copy(localFile, remoteFile); err != nil {
		return fmt.Errorf("ssh sftp: 下载文件失败: %w", err)
	}
	return nil
}

// === 目录传输

// UploadDir 递归上传本地目录到远程
func (s *SFTP) UploadDir(localDir, remoteDir string) error {
	return filepath.Walk(localDir, func(localPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		remotePath := filepath.ToSlash(filepath.Join(remoteDir, localPath[len(localDir):]))

		if info.IsDir() {
			if err := s.client.MkdirAll(remotePath); err != nil {
				return fmt.Errorf("ssh sftp: 创建远程目录 %s 失败: %w", remotePath, err)
			}
		} else {
			if err := s.Upload(localPath, remotePath); err != nil {
				return err
			}
		}
		return nil
	})
}

// DownloadDir 递归下载远程目录到本地
func (s *SFTP) DownloadDir(remoteDir, localDir string) error {
	if err := os.MkdirAll(localDir, 0775); err != nil {
		return fmt.Errorf("ssh sftp: 创建本地目录失败: %w", err)
	}

	entries, err := s.client.ReadDir(remoteDir)
	if err != nil {
		return fmt.Errorf("ssh sftp: 读取远程目录失败: %w", err)
	}

	for _, entry := range entries {
		remotePath := filepath.ToSlash(filepath.Join(remoteDir, entry.Name()))
		localPath := filepath.ToSlash(filepath.Join(localDir, entry.Name()))

		if entry.IsDir() {
			if err := s.DownloadDir(remotePath, localPath); err != nil {
				continue
			}
		} else {
			if err := s.Download(remotePath, localPath); err != nil {
				continue
			}
		}
	}
	return nil
}
