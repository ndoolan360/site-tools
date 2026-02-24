package sitetools

import (
	"bytes"
	"io/fs"
	"path"
	"strings"
)

type Build struct {
	Assets
}

func (build *Build) FromDir(fsys fs.FS, root string) error {
	return build.walkDir(fsys, root, false)
}

func (build *Build) walkDir(fsys fs.FS, root string, includeRoot bool) error {
	return fs.WalkDir(fsys, root,
		func(filepath string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			data, err := fs.ReadFile(fsys, filepath)
			if err != nil {
				return err
			}

			var assetPath string
			if includeRoot {
				assetPath = "/" + filepath
			} else {
				assetPath = strings.TrimPrefix(filepath, root)
			}
			assetPath = path.Clean("/" + strings.TrimPrefix(assetPath, "/"))

			build.Assets = append(build.Assets, &Asset{
				Path: assetPath,
				Meta: map[string]any{},
				Data: bytes.TrimSpace(data),
			})

			return nil
		},
	)
}
