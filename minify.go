package sitetools

import (
	"mime"
	"path"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/svg"
)

type MinifyTransformer struct {
	minifier *minify.M
}

func (m *MinifyTransformer) getMinifier() *minify.M {
	if m.minifier == nil {
		m.minifier = minify.New()
		m.minifier.Add("text/html", &html.Minifier{KeepEndTags: true})
		m.minifier.Add("text/css", &css.Minifier{})
		m.minifier.Add("text/javascript", &js.Minifier{})
		m.minifier.Add("image/svg+xml", &svg.Minifier{})
	}

	return m.minifier
}

func (m *MinifyTransformer) Transform(asset *Asset) error {
	fileType := path.Ext(asset.Path)
	mimeType := mime.TypeByExtension(fileType)

	if minified, err := m.getMinifier().Bytes(mimeType, asset.Data); err == nil {
		asset.Data = minified
	}

	return nil
}
