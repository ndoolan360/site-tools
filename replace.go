package sitetools

import "bytes"

type ReplacerTransformer struct {
	Replacements map[string]string
}

func (t ReplacerTransformer) Transform(asset *Asset) error {
	for key, value := range t.Replacements {
		asset.Data = bytes.ReplaceAll(asset.Data, []byte(key), []byte(value))
	}
	return nil
}
