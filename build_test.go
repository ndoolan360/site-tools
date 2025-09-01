package sitetools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWalkDir(t *testing.T) {
	t.Run("WalkDir", func(t *testing.T) {
		includeRootStates := []bool{true, false}
		for _, includeRoot := range includeRootStates {
			dir := t.TempDir()
			// create root directory
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				t.Fatalf("failed to create root directory: %v", err)
			}

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

			err = build.walkDir(os.DirFS(dir), ".", includeRoot)
			if err != nil {
				t.Fatalf("WalkDir failed: %v", err)
			}

			// Check if the asset was added correctly
			if len(build.Assets) != 3 {
				t.Fatalf("expected 3 assets, got %d", len(build.Assets))
			}
			if string(build.Assets[0].Data) != "test content" {
				t.Fatalf("expected 'test content', got '%s'", string(build.Assets[0].Data))
			}
		}
	})
}
