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

// TestFromDir_SecondCallOverlaysFirst verifies that calling FromDir a
// second time over a directory with an overlapping file overrides that
// file's content rather than adding a duplicate asset.
func TestFromDir_SecondCallOverlaysFirst(t *testing.T) {
	baseDir := t.TempDir()
	overlayDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(baseDir, "shared.txt"), []byte("base"), 0644); err != nil {
		t.Fatalf("failed to create base file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "base-only.txt"), []byte("base-only"), 0644); err != nil {
		t.Fatalf("failed to create base-only file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(overlayDir, "shared.txt"), []byte("overlay"), 0644); err != nil {
		t.Fatalf("failed to create overlay file: %v", err)
	}

	build := &Build{}
	if err := build.FromDir(os.DirFS(baseDir), "."); err != nil {
		t.Fatalf("FromDir(base) failed: %v", err)
	}
	if err := build.FromDir(os.DirFS(overlayDir), "."); err != nil {
		t.Fatalf("FromDir(overlay) failed: %v", err)
	}

	if len(build.Assets) != 2 {
		t.Fatalf("expected 2 assets after overlaying a shared path, got %d", len(build.Assets))
	}

	shared := build.Assets.Pop(WithPath("/shared.txt"))
	if len(shared) != 1 {
		t.Fatalf("expected exactly one asset at /shared.txt, got %d", len(shared))
	}
	if string(shared[0].Data) != "overlay" {
		t.Errorf("Asset Data = %s, want overlay", string(shared[0].Data))
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
