package sitetools

import (
	"bytes"
	"fmt"
	"path"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
)

func (build *Build) FromGit(url string, branch string, outDir string) error {
	fsys := memfs.New()

	_, err := git.Clone(memory.NewStorage(), fsys, &git.CloneOptions{
		URL:               url,
		RecurseSubmodules: git.NoRecurseSubmodules,
		SingleBranch:      true,
		ReferenceName:     plumbing.NewBranchReferenceName(branch),
	})
	if err != nil && err != git.ErrRepositoryAlreadyExists {
		return fmt.Errorf("could not clone repository: %v", err)
	}

	return build.walkBilly(fsys, ".", outDir)
}

func (build *Build) walkBilly(fsys billy.Filesystem, root string, prefix string) error {
	entries, err := fsys.ReadDir(root)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		name := entry.Name()
		fullPath := name
		if root != "." && root != "" && root != "/" {
			fullPath = path.Join(root, name)
		}

		if entry.IsDir() {
			if err := build.walkBilly(fsys, fullPath, prefix); err != nil {
				return err
			}
			continue
		}

		data, err := util.ReadFile(fsys, fullPath)
		if err != nil {
			return err
		}

		assetPath := path.Join("/", prefix, fullPath)
		build.Assets = append(build.Assets, &Asset{
			Path: assetPath,
			Meta: map[string]any{},
			Data: bytes.TrimSpace(data),
		})
	}

	return nil
}
