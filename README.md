# site-tools

A small, composable static site builder written in Go. It loads assets from a directory or Git, parses front matter, converts Markdown to HTML with code highlighting, applies templates/components, runs replacements and minification, and writes the result—plus optional auto‑reload for local dev.

## A brief example

```go
package main

import (
	"os"

	"github.com/ndoolan360/site-tools"
)

func main() {
	build := sitetools.Build{}
	_ = build.FromDir(os.DirFS("."), "content")

	_ = build.Assets.Transform(
		sitetools.CollectFrontMatter{},
		sitetools.MarkdownTransformer{},
		sitetools.TemplateTransformer{Global: map[string]any{"SiteName": "My Site"}},
		sitetools.MinifyTransformer{},
	)

	_ = build.AddSitemap("https://example.com")
	_ = build.Assets.Write("public")
}
```
