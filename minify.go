package sitetools

import (
	"fmt"
	"mime"
	"path"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/svg"
	"github.com/tdewolff/minify/v2/xml"
)

// MinifyTransformer minifies asset contents based on their file extension.
//
// The HTML, CSS, JS, SVG, and XML fields expose the underlying minifier
// options for each supported content type, allowing callers to customize
// minification behavior. Zero values use each minifier's defaults.
type MinifyTransformer struct {
	HTML html.Minifier
	CSS  css.Minifier
	JS   js.Minifier
	SVG  svg.Minifier
	XML  xml.Minifier
}

func (m MinifyTransformer) buildMinifier() *minify.M {
	mm := minify.New()
	mm.Add("text/html", &m.HTML)
	mm.Add("text/css", &m.CSS)
	mm.Add("text/javascript", &m.JS)
	mm.Add("image/svg+xml", &m.SVG)
	mm.Add("application/xml", &m.XML)
	mm.Add("text/xml", &m.XML)
	return mm
}

func (m MinifyTransformer) Transform(asset *Asset) error {
	fileType := path.Ext(asset.Path)
	mimeType := mime.TypeByExtension(fileType)

	if mimeType == "" {
		return nil
	}

	minified, err := m.buildMinifier().Bytes(mimeType, asset.Data)
	if err != nil {
		if err == minify.ErrNotExist {
			return nil
		}
		return fmt.Errorf("minify failed for %s (%s): %w", asset.Path, mimeType, err)
	}

	asset.Data = minified

	return nil
}
