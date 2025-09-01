package sitetools

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func (build *Build) FromGit(url string, branch string, outDir string) error {
	root := "tmp/" + outDir

	_, err := git.PlainClone(root, false, &git.CloneOptions{
		URL:               url,
		RecurseSubmodules: git.NoRecurseSubmodules,
		SingleBranch:      true,
		ReferenceName:     plumbing.NewBranchReferenceName(branch),
	})
	if err != nil && err != git.ErrRepositoryAlreadyExists {
		return fmt.Errorf("could not clone repository: %v", err)
	}

	return build.walkDir(os.DirFS("tmp"), outDir, true)
}
