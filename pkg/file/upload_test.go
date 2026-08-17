package file

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestFileUploadAllowExts(t *testing.T) {
	dir := t.TempDir()
	cfg := FileConfig{
		Dir:       dir,
		MaxSize:   1,
		AllowExts: []string{".PNG"},
	}

	data := []byte("image data")
	path, err := (FileUpload{
		Name: "photo.png",
		Data: &data,
	}).Save(cfg)
	if err != nil {
		t.Fatalf("Save allowed extension: %v", err)
	}
	if filepath.Ext(path) != ".png" {
		t.Fatalf("saved extension = %q, want .png", filepath.Ext(path))
	}
	if filepath.ToSlash(path) != path || strings.Contains(path, `\`) {
		t.Fatalf("saved path = %q, want slash-separated path", path)
	}

	_, err = (FileUpload{
		Name: "document.txt",
		Data: &data,
	}).Save(cfg)
	if err == nil || !strings.Contains(err.Error(), "unsupported file extension") {
		t.Fatalf("Save rejected extension error = %v, want unsupported file extension", err)
	}
}

func TestUploadMergeReturnsCanonicalPath(t *testing.T) {
	cfg := FileConfig{Dir: filepath.ToSlash(t.TempDir()), MaxSize: 1}
	for index := range 2 {
		data := []byte{byte(index)}
		upload := FileUpload{Id: "upload-id", Index: strconv.Itoa(index), Data: &data}
		if err := upload.ChunkSave(cfg); err != nil {
			t.Fatalf("ChunkSave(%d): %v", index, err)
		}
	}

	path, err := cfg.ChunkMerge("upload-id", "merged.bin")
	if err != nil {
		t.Fatalf("ChunkMerge: %v", err)
	}
	if filepath.ToSlash(path) != path || strings.Contains(path, `\`) {
		t.Fatalf("merged path = %q, want slash-separated path", path)
	}
	if _, err := os.Stat(filepath.FromSlash(path)); err != nil {
		t.Fatalf("Stat merged path: %v", err)
	}
}
