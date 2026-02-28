package sitetools

import (
	"mime"
	"path"
	"slices"
	"strings"
)

type Filter func(Asset) bool

func (assets *Assets) Pop(filters ...Filter) Assets {
	if len(filters) == 0 {
		return nil
	}

	keep := make(Assets, 0, len(*assets))
	pop := make(Assets, 0, len(*assets))

	for _, asset := range *assets {
		willPop := true
		for _, filter := range filters {
			if !filter(*asset) {
				keep = append(keep, asset)
				willPop = false
				break
			}
		}
		if willPop {
			pop = append(pop, asset)
		}
	}

	*assets = keep
	return pop
}

func (assets Assets) Filter(filters ...Filter) Assets {
	if len(filters) == 0 {
		return assets
	}

	return assets.Pop(filters...)
}

func WithParentDir(parent string) Filter {
	return func(asset Asset) bool {
		dir := path.Dir(asset.Path)
		if path.Base(asset.Path) == path.Base(parent) && dir == path.Dir(parent) {
			return true
		} else if dir == "/" || dir == "." {
			return false
		} else {
			return WithParentDir(parent)(Asset{Path: dir})
		}
	}
}

func WithoutParentDir(parent string) Filter {
	return func(asset Asset) bool {
		return !WithParentDir(parent)(asset)
	}
}

func WithPath(filepath string) Filter {
	return func(asset Asset) bool {
		return path.Clean(filepath) == path.Clean(asset.Path)
	}
}

func WithoutPath(filepath string) Filter {
	return func(asset Asset) bool {
		return !WithPath(filepath)(asset)
	}
}

func WithExtensions(exts ...string) Filter {
	return func(asset Asset) bool {
		ext := path.Ext(asset.Path)
		return slices.Contains(exts, ext)
	}
}

func WithoutExtensions(exts ...string) Filter {
	return func(asset Asset) bool {
		return !WithExtensions(exts...)(asset)
	}
}

func WithMeta(key string) Filter {
	return func(asset Asset) bool {
		val, ok := asset.Meta[key]
		if ok {
			switch v := val.(type) {
			case bool:
				return v
			case string:
				normalized := strings.ToLower(strings.TrimSpace(v))
				if normalized == "true" {
					return true
				}
				if normalized == "false" {
					return false
				}
			}
			return val != false
		}
		return false
	}
}

func WithoutMeta(key string) Filter {
	return func(asset Asset) bool {
		return !WithMeta(key)(asset)
	}
}

// WithMimeType returns a filter that matches assets with the given MIME types.
// The MIME types can be specified as full MIME types (e.g. "text/css") or top-level
// types with a wildcard (e.g. "image/*").
func WithMimeType(mimeTypes ...string) Filter {
	mime.AddExtensionType(".md", "text/markdown")
	mime.AddExtensionType(".yaml", "application/yaml")
	mime.AddExtensionType(".yml", "application/yaml")

	return func(asset Asset) bool {
		ext := path.Ext(asset.Path)
		assetMimeType := mime.TypeByExtension(ext)

		// remove charset if present
		if idx := strings.Index(assetMimeType, ";"); idx != -1 {
			assetMimeType = strings.TrimSpace(assetMimeType[:idx])
		}

		for _, mimeType := range mimeTypes {
			if ext != "" {
				fullMatch := mimeType == assetMimeType
				topLevelMatch := strings.HasSuffix(mimeType, "/*") &&
					strings.HasPrefix(assetMimeType, strings.TrimSuffix(mimeType, "*"))
				if fullMatch || topLevelMatch {
					return true
				}
			}
		}
		return false
	}
}

func WithoutMimeType(mimeTypes ...string) Filter {
	return func(asset Asset) bool {
		return !WithMimeType(mimeTypes...)(asset)
	}
}
