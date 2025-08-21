package builder

import (
	"bytes"
	"maps"

	"github.com/adrg/frontmatter"
)

type CollectFrontMatter struct{}

func (p CollectFrontMatter) Transform(asset *Asset) error {
	var meta map[string]any

	rest, err := frontmatter.Parse(bytes.NewReader(asset.Data), &meta)
	if err != nil {
		return err
	}

	if asset.Meta == nil {
		asset.Meta = make(map[string]any)
	}

	maps.Copy(asset.Meta, meta)
	asset.Data = rest

	return nil
}
