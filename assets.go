package sitetools

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Asset struct {
	Path string
	Data []byte
	Meta map[string]any
}

type Assets []*Asset

func (assets Assets) Transform(transformers ...Transformer) error {
	for _, transformer := range transformers {
		for _, asset := range assets {
			if err := transformer.Transform(asset); err != nil {
				return err
			}
		}
	}
	return nil
}

func (assets Assets) Write(outDir string) error {
	baseDir, err := filepath.Abs(outDir)
	if err != nil {
		return err
	}

	for _, asset := range assets {
		cleaned := path.Clean("/" + strings.TrimPrefix(asset.Path, "/"))
		rel := strings.TrimPrefix(cleaned, "/")

		target := filepath.Join(baseDir, filepath.FromSlash(rel))
		targetAbs, err := filepath.Abs(target)
		if err != nil {
			return err
		}
		if targetAbs != baseDir && !strings.HasPrefix(targetAbs, baseDir+string(os.PathSeparator)) {
			return fmt.Errorf("asset path escapes output dir: %s", asset.Path)
		}

		if err := os.MkdirAll(filepath.Dir(targetAbs), 0755); err != nil {
			return err
		}

		if err := os.WriteFile(targetAbs, asset.Data, 0644); err != nil {
			return err
		}
	}

	return nil
}

func (assets Assets) ToMap(keyFromMeta string) map[string]*Asset {
	m := make(map[string]*Asset)
	for _, asset := range assets {
		if key, ok := asset.Meta[keyFromMeta].(string); ok {
			m[key] = asset
		}
	}
	return m
}

func (assets Assets) SetMetaFunc(metaKey string, fn func(Asset) string) Assets {
	for _, asset := range assets {
		// If the meta is nil, skip
		if asset.Meta == nil {
			continue
		}

		asset.Meta[metaKey] = fn(*asset)
	}

	return assets
}

func (assets Assets) AddToMeta(metaKey string, value string) Assets {
	return assets.SetMetaFunc(metaKey, func(asset Asset) string {
		return value
	})
}
