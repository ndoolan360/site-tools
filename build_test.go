package sitetools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFromDir(t *testing.T) {
	dir := t.TempDir()

	testFiles := []string{
		"test1.txt",
		"test2.txt",
		"test3.txt",
	}
	for _, file := range testFiles {
		testFilePath := filepath.Join(dir, file)
		err := os.WriteFile(testFilePath, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	build := &Build{}

	err := build.FromDir(os.DirFS(dir), ".")
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}

	// Check if the asset was added correctly
	if len(build.Assets) != 3 {
		t.Fatalf("expected 3 assets, got %d", len(build.Assets))
	}
	if string(build.Assets[0].Data) != "test content" {
		t.Fatalf("expected 'test content', got '%s'", string(build.Assets[0].Data))
	}
}

func TestFromDir_ReadFileError(t *testing.T) {
	dir := t.TempDir()

	testFilePath := filepath.Join(dir, "test1.txt")
	err := os.WriteFile(testFilePath, []byte("test content"), 0000) // no permissions
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	build := &Build{}

	err = build.FromDir(os.DirFS(dir), ".")
	if err == nil {
		t.Fatal("expected error due to file read permissions, got nil")
	}
}

func TestWalkDir_IncludeRoot(t *testing.T) {
	dir := t.TempDir()

	testFilePath := filepath.Join(dir, "test1.txt")
	err := os.WriteFile(testFilePath, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	build := &Build{}

	err = build.walkDir(os.DirFS(dir), ".", true)
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}

	if len(build.Assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(build.Assets))
	}

	if build.Assets[0].Path != "/test1.txt" {
		t.Fatalf("expected path '/test1.txt', got '%s'", build.Assets[0].Path)
	}
}
