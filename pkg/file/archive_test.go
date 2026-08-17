package file

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchiveTarget(t *testing.T) {
	if _, err := archiveTarget(".", "safe.txt"); err != nil {
		t.Fatalf("safe entry rejected: %v", err)
	}
	if _, err := archiveTarget(".", "../escape.txt"); err == nil {
		t.Fatal("escaping entry unexpectedly accepted")
	}
}

func TestZipUnpackRelativeRoot(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	zipPath := filepath.Join(dir, "test.zip")

	output, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(output)
	writer, err := zw.CreateHeader(&zip.FileHeader{Name: "safe.txt", Method: zip.Deflate})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("safe")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}

	if err := ZipUnpack(zipPath, "."); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "safe.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "safe" {
		t.Fatalf("content = %q, want safe", got)
	}
}

func TestZipUnpackRejectsEscape(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "test.zip")
	output, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(output)
	writer, err := zw.CreateHeader(&zip.FileHeader{Name: "../escape.txt", Method: zip.Deflate})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("escaped")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}

	err = ZipUnpack(zipPath, dir)
	if err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("ZipUnpack error = %v, want outside error", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "escape.txt")); !os.IsNotExist(err) {
		t.Fatalf("escape file exists or stat returned error %v", err)
	}
}
