package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUploadRejectsPathEscape(t *testing.T) {
	root := t.TempDir()
	cfg := FileConfig{Dir: filepath.Join(root, "upload")}
	data := []byte("data")
	upload := FileUpload{Data: &data}

	invalidIDs := []string{"", ".", "..", "../outside", `..\outside`, "a/b", `a\b`}
	for _, id := range invalidIDs {
		upload.Id = id
		upload.Index = "0"
		if err := upload.ChunkSave(cfg); err == nil {
			t.Fatalf("ChunkSave id %q unexpectedly succeeded", id)
		}
	}

	invalidIndexes := []string{"", "-1", "x", "../0", `..\0`, "1.0"}
	for _, index := range invalidIndexes {
		upload.Id = "upload-id"
		upload.Index = index
		if err := upload.ChunkSave(cfg); err == nil {
			t.Fatalf("ChunkSave index %q unexpectedly succeeded", index)
		}
	}
}

func TestUploadSaveValidatesInputAndSanitizesPath(t *testing.T) {
	dir := t.TempDir()
	cfg := FileConfig{Dir: dir}

	if _, err := (FileUpload{Name: "empty.txt"}).Save(cfg); err == nil {
		t.Fatal("Save without Reader or Data unexpectedly succeeded")
	}

	data := []byte("content")
	path, err := (FileUpload{Name: "my file.txt", Data: &data}).Save(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if rel, relErr := filepath.Rel(dir, path); relErr != nil || rel == "" || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		t.Fatalf("saved path %s escapes %s", path, dir)
	}
	if strings.Contains(filepath.Base(path), " ") {
		t.Fatalf("saved base name still contains a space: %s", filepath.Base(path))
	}
}

func TestUploadChunkOrderAndMerge(t *testing.T) {
	root := t.TempDir()
	cfg := FileConfig{Dir: filepath.Join(root, "upload")}
	chunks := map[string]string{
		"0":  "A",
		"1":  "B",
		"2":  "C",
		"10": "D",
	}
	for index, content := range chunks {
		data := []byte(content)
		upload := FileUpload{Id: "file", Index: index, Data: &data}
		if err := upload.ChunkSave(cfg); err != nil {
			t.Fatal(err)
		}
	}

	names, err := cfg.ChunkList("file")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(names, ",") != "0,1,2,10" {
		t.Fatalf("chunk order = %v, want [0 1 2 10]", names)
	}

	path, err := cfg.ChunkMerge("file", "result.txt")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ABCD" {
		t.Fatalf("merged content = %q, want ABCD", got)
	}
	if _, err := os.Stat(filepath.Join(cfg.Dir, "file")); !os.IsNotExist(err) {
		t.Fatalf("chunk directory still exists or stat returned error %v", err)
	}
}

func TestUploadMergeValidatesExtension(t *testing.T) {
	cfg := FileConfig{
		Dir:       t.TempDir(),
		AllowExts: []string{".png"},
	}
	if _, err := cfg.ChunkMerge("id", "result.txt"); err == nil || !strings.Contains(err.Error(), "unsupported file extension") {
		t.Fatalf("ChunkMerge error = %v, want unsupported file extension", err)
	}
}

func TestReadStreamEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.bin")
	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ReadStream(path, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Data) != 0 || result.Start != 0 || result.End != -1 || result.Total != 0 {
		t.Fatalf("empty result = %+v", result)
	}
}

func TestListDirReturnsInvalidPatternError(t *testing.T) {
	_, err := ListDir(t.TempDir(), "[")
	if err == nil {
		t.Fatal("ListDir with invalid pattern unexpectedly succeeded")
	}
}

func TestCopyFileSameFileIsNoop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "same.txt")
	if err := os.WriteFile(path, []byte("unchanged"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CopyFile(path, path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "unchanged" {
		t.Fatalf("same-file copy changed content to %q", got)
	}
}
