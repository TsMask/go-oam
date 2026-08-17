package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const pathLockCount = 64

var pathLocks [pathLockCount]sync.Mutex

// withPathLock serializes operations targeting the same path in the current
// process. A fixed lock table avoids retaining one mutex per historical path;
// hash collisions only cause unrelated paths to wait briefly.
func withPathLock(path string, fn func() error) error {
	key := filepath.Clean(path)
	if abs, err := filepath.Abs(path); err == nil {
		key = filepath.Clean(abs)
	}
	key = strings.ToLower(key)

	var hash uint32 = 2166136261
	for i := range len(key) {
		hash ^= uint32(key[i])
		hash *= 16777619
	}
	mu := &pathLocks[hash%pathLockCount]
	mu.Lock()
	defer mu.Unlock()
	return fn()
}

// writeFileAtomic writes dst through a unique temporary file and replaces the
// destination only after write and close both succeed.
func writeFileAtomic(dst string, perm os.FileMode, write func(*os.File) error) error {
	dir, base := filepath.Split(dst)
	if base == "" {
		return fmt.Errorf("invalid destination path %q", dst)
	}
	if dir == "" {
		dir = "."
	}

	f, err := os.CreateTemp(dir, "."+base+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary file for %s: %w", dst, err)
	}
	tmp := f.Name()

	writeErr := write(f)
	closeErr := f.Close()
	if writeErr == nil {
		writeErr = closeErr
	}
	if writeErr == nil {
		writeErr = os.Chmod(tmp, perm.Perm())
	}
	if writeErr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("write %s: %w", dst, writeErr)
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace %s: %w", dst, err)
	}
	return nil
}

// archiveTarget joins an archive entry to root and rejects entries that escape
// root. filepath.Rel is used instead of a string prefix so "." and other clean
// relative roots work correctly.
func archiveTarget(root, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("empty archive entry name")
	}

	cleanRoot := filepath.Clean(root)
	target := filepath.Join(cleanRoot, filepath.Clean(name))
	rel, err := filepath.Rel(cleanRoot, target)
	if err != nil {
		return "", fmt.Errorf("archive entry %q outside %s: %w", name, root, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive entry %q outside %s", name, root)
	}
	return target, nil
}

// canonicalPath returns the package's public path format. Slash-separated
// paths are accepted by the OS on every supported platform and avoid exposing
// runtime-specific separators to callers.
func canonicalPath(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
}
