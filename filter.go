package builder

import "path"

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

func WithPath(filepath string) Filter {
	return func(asset Asset) bool {
		return path.Clean(filepath) == path.Clean(asset.Path)
	}
}

func WithoutPath(filepath string) Filter {
	return func(asset Asset) bool {
		return path.Clean(filepath) != path.Clean(asset.Path)
	}
}

func WithExtensions(exts ...string) Filter {
	return func(asset Asset) bool {
		for _, ext := range exts {
			if path.Ext(asset.Path) == ext {
				return true
			}
		}
		return false
	}
}

func WithoutExtensions(exts ...string) Filter {
	return func(asset Asset) bool {
		for _, ext := range exts {
			if path.Ext(asset.Path) == ext {
				return false
			}
		}
		return true
	}
}

func WithMeta(key string) Filter {
	return func(asset Asset) bool {
		val, ok := asset.Meta[key]
		if ok {
			return val != false
		}
		return false
	}
}

func WithoutMeta(key string) Filter {
	return func(asset Asset) bool {
		val, ok := asset.Meta[key]
		if ok {
			return val == false
		}
		return true
	}
}
