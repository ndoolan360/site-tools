package sitetools

import (
	"mime"
	"path"
	"sync"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/svg"
	"github.com/tdewolff/minify/v2/xml"
)

type MinifyTransformer struct{}

var sharedMinifier *minify.M
var sharedMinifierOnce sync.Once

func getMinifier() *minify.M {
	sharedMinifierOnce.Do(func() {
		sharedMinifier = minify.New()
		sharedMinifier.Add("text/html", &html.Minifier{KeepEndTags: true})
		sharedMinifier.Add("text/css", &css.Minifier{})
		sharedMinifier.Add("text/javascript", &js.Minifier{})
		sharedMinifier.Add("image/svg+xml", &svg.Minifier{})
		sharedMinifier.Add("application/xml", &xml.Minifier{})
		sharedMinifier.Add("text/xml", &xml.Minifier{})
	})

	return sharedMinifier
}

func (m MinifyTransformer) Transform(asset *Asset) error {
	fileType := path.Ext(asset.Path)
	mimeType := mime.TypeByExtension(fileType)

	if minified, err := getMinifier().Bytes(mimeType, asset.Data); err == nil {
		asset.Data = minified
	}

	return nil
}
