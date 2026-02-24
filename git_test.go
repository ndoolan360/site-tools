package sitetools

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestFromGit_SuccessfulClone(t *testing.T) {
	files := map[string]string{
		"file1.txt":        "content1",
		"subdir/file2.txt": "content2",
	}
	repoPath, cleanupRepo := setupGitRepo(t, files, "master")
	defer cleanupRepo()

	buildInstance := &Build{}
	outDir := "test-clone-main"

	err := buildInstance.FromGit(repoPath, "master", outDir)
	if err != nil {
		t.Fatalf("FromGit failed: %v", err)
	}

	expectedAssets := map[string]string{
		"/test-clone-main/file1.txt":        "content1",
		"/test-clone-main/subdir/file2.txt": "content2",
	}

	verifyAssets(t, buildInstance.Assets, expectedAssets, outDir)
}

func TestFromGit_InvalidRepositoryURL(t *testing.T) {
	buildInstance := &Build{}
	outDir := "test-clone-invalid-url"

	err := buildInstance.FromGit("://invalid-url-format", "master", outDir)
	if err == nil {
		t.Fatalf("Expected FromGit to fail for invalid URL, but it succeeded")
	}
	expectedErrorSubString := "could not clone repository"
	if !strings.Contains(err.Error(), expectedErrorSubString) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedErrorSubString, err)
	}
}

func TestFromGit_OutDirWithSubdirectories(t *testing.T) {
	files := map[string]string{"root_file.txt": "root_content"}
	repoPath, cleanupRepo := setupGitRepo(t, files, "master")
	defer cleanupRepo()

	buildInstance := &Build{}
	outDir := "parentDir/clonedRepo"

	err := buildInstance.FromGit(repoPath, "master", outDir)
	if err != nil {
		t.Fatalf("FromGit failed with subpath outDir: %v", err)
	}

	verifyAssets(t, buildInstance.Assets, map[string]string{"/parentDir/clonedRepo/root_file.txt": "root_content"}, outDir)
}

func TestWalkBilly_ReadDirError(t *testing.T) {
	fsys := memfs.New()

	buildInstance := &Build{}
	err := buildInstance.walkBilly(fsys, "missing", "prefix")
	if err == nil {
		t.Fatal("Expected error when root is missing, but got nil")
	}
}

func TestWalkBilly_PrefixAndRoot(t *testing.T) {
	fsys := memfs.New()
	if err := fsys.MkdirAll("subdir", 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	if err := util.WriteFile(fsys, "subdir/file.txt", []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	buildInstance := &Build{}
	err := buildInstance.walkBilly(fsys, "subdir", "prefix")
	if err != nil {
		t.Fatalf("walkBilly failed: %v", err)
	}

	if len(buildInstance.Assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(buildInstance.Assets))
	}
	if buildInstance.Assets[0].Path != "/prefix/subdir/file.txt" {
		t.Fatalf("expected asset path '/prefix/subdir/file.txt', got '%s'", buildInstance.Assets[0].Path)
	}
	if string(buildInstance.Assets[0].Data) != "content" {
		t.Fatalf("expected asset data 'content', got '%s'", string(buildInstance.Assets[0].Data))
	}
}

func setupGitRepo(t *testing.T, files map[string]string, initialBranchName string) (repoPath string, cleanupFunc func()) {
	t.Helper()

	var err error
	repoPath = t.TempDir()
	cleanupFunc = func() {}

	r, err := git.PlainInit(repoPath, false) // false for not bare
	if err != nil {
		cleanupFunc()
		t.Fatalf("Failed to init git repo at %s: %v", repoPath, err)
	}

	w, err := r.Worktree()
	if err != nil {
		cleanupFunc()
		t.Fatalf("Failed to get worktree: %v", err)
	}

	for filePath, content := range files {
		fullPath := filepath.Join(repoPath, filePath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			cleanupFunc()
			t.Fatalf("Failed to create dir for file %s: %v", filePath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			cleanupFunc()
			t.Fatalf("Failed to write file %s: %v", filePath, err)
		}
		_, err = w.Add(filePath)
		if err != nil {
			cleanupFunc()
			t.Fatalf("Failed to add file %s to worktree: %v", filePath, err)
		}
	}

	commitMsg := "Initial commit"
	_, err = w.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})

	// Set up the initial branch
	initialBranchRefName := plumbing.NewBranchReferenceName(initialBranchName)
	currentHead, err := r.Head()
	if err != nil {
		cleanupFunc()
		t.Fatalf("Failed to get current HEAD: %v", err)
	}

	if currentHead.Name() != initialBranchRefName {
		t.Fatal("Current HEAD is not the initial branch")
	}

	return repoPath, cleanupFunc
}

// Helper to verify assets in tests
func verifyAssets(t *testing.T, assets Assets, expectedAssets map[string]string, outDir string) {
	t.Helper()

	// filter out .git directory
	assets = assets.Filter(func(asset Asset) bool {
		return !strings.HasPrefix(asset.Path, path.Join("/", outDir, ".git"))
	})

	if len(assets) != len(expectedAssets) {
		mapKeys := make([]string, 0, len(expectedAssets))
		for k := range expectedAssets {
			mapKeys = append(mapKeys, k)
		}

		t.Errorf("Expected %d assets, got %d. Got paths: %v, Expected paths: %v",
			len(expectedAssets), len(assets), assets, mapKeys)
		return
	}

	foundAssets := make(map[string]string)
	for _, asset := range assets {
		foundAssets[asset.Path] = string(asset.Data)
	}

	for expectedPath, expectedContent := range expectedAssets {
		actualContent, ok := foundAssets[expectedPath]
		if !ok {
			t.Errorf("Expected asset path '%s' not found. Found assets: %v", expectedPath, assets)
			continue
		}
		if actualContent != expectedContent {
			t.Errorf("Asset '%s' content mismatch: got '%s', want '%s'", expectedPath, actualContent, expectedContent)
		}
	}
}
