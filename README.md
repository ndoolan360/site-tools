# site-tools

A composable static site builder for Go. Load assets, apply transformers, and write the result. Import only the feature packages you need.

## Usage

```go
package main

import (
	"log"
	"os"

	sitetools "go.doolan.dev/site-tools"
	"go.doolan.dev/site-tools/frontmatter"
	"go.doolan.dev/site-tools/markdown"
	"go.doolan.dev/site-tools/minify"
)

func main() {
	build := sitetools.Build{}
	if err := build.FromDir(os.DirFS("."), "content"); err != nil {
		log.Fatal(err)
	}

	if err := build.Assets.Filter(sitetools.WithExtensions(".md")).Transform(
		frontmatter.CollectFrontMatter{},
		markdown.MarkdownTransformer{},
		minify.MinifyTransformer{},
	); err != nil {
		log.Fatal(err)
	}

	if err := build.Assets.Write("public"); err != nil {
		log.Fatal(err)
	}
}
```

Transformers run in the order supplied. `Filter` selects assets without removing them from the build.

## Packages

| Package | Description |
| --- | --- |
| [`go.doolan.dev/site-tools`](https://pkg.go.dev/go.doolan.dev/site-tools) | Core assets, build pipelines, filters, templates, and replacements. |
| [`go.doolan.dev/site-tools/encryption`](https://pkg.go.dev/go.doolan.dev/site-tools/encryption) | Page encryption with AES-GCM and browser-side decryption. |
| [`go.doolan.dev/site-tools/encryption/age`](https://pkg.go.dev/go.doolan.dev/site-tools/encryption/age) | Optional age encryption backend with bundled browser decryption. |
| [`go.doolan.dev/site-tools/frontmatter`](https://pkg.go.dev/go.doolan.dev/site-tools/frontmatter) | Front-matter parsing into asset metadata. |
| [`go.doolan.dev/site-tools/images`](https://pkg.go.dev/go.doolan.dev/site-tools/images) | Image transcoding and SVG inlining. |
| [`go.doolan.dev/site-tools/markdown`](https://pkg.go.dev/go.doolan.dev/site-tools/markdown) | Markdown rendering with syntax highlighting. |
| [`go.doolan.dev/site-tools/minify`](https://pkg.go.dev/go.doolan.dev/site-tools/minify) | HTML, CSS, JavaScript, SVG, and XML minification. |
| [`go.doolan.dev/site-tools/publishing`](https://pkg.go.dev/go.doolan.dev/site-tools/publishing) | Sitemap and robots.txt generation, and comment prefixes. |

The core, encryption, and publishing packages use only the standard library. All packages share one `go.mod`.

## Tests

```sh
go test ./...
```
