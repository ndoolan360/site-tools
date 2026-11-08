package frontmatter

import (
	"bytes"
	"fmt"
	"maps"

	"github.com/adrg/frontmatter"
	sitetools "go.doolan.dev/site-tools"
)

type CollectFrontMatter struct{}

var _ sitetools.Transformer = CollectFrontMatter{}

func (p CollectFrontMatter) Transform(asset *sitetools.Asset) error {
	var meta map[string]any

	rest, err := frontmatter.Parse(bytes.NewReader(asset.Data), &meta)
	if err != nil {
		return fmt.Errorf("issue in asset %s: %w", asset.Path, err)
	}

	if asset.Meta == nil {
		asset.Meta = make(map[string]any)
	}

	maps.Copy(asset.Meta, meta)
	asset.Data = rest

	return nil
}
