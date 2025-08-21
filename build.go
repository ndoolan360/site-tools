package builder

import (
	"bytes"
	"io/fs"
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

			if includeRoot {
				filepath = "/" + filepath
			} else {
				filepath = strings.TrimPrefix(filepath, root)
			}

			build.Assets = append(build.Assets, &Asset{
				Path: filepath,
				Meta: map[string]any{},
				Data: bytes.TrimSpace(data),
			})

			return nil
		},
	)
}
