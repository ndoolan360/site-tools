package sitetools

import (
	"bytes"
	"path"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	fences "github.com/stefanfritsch/goldmark-fences"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	goldmark_parser "github.com/yuin/goldmark/parser"
	goldmark_renderer "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

type Extensions = []goldmark.Extender
type ParserOptions = []goldmark_parser.Option
type RenderOptions = []goldmark_renderer.Option

type MarkdownTransformer struct {
	Extensions
	ParserOptions
	RenderOptions
}

func (p MarkdownTransformer) newGoldmark() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(
			append(
				p.Extensions,
				extension.GFM,
				&fences.Extender{},
				extension.Typographer,
				meta.Meta,
				highlighting.NewHighlighting(
					highlighting.WithWrapperRenderer(codeWrapperRenderer),
					highlighting.WithFormatOptions(
						chromahtml.WithClasses(true),
						chromahtml.WithLineNumbers(true),
						chromahtml.LineNumbersInTable(true),
					),
				),
			)...,
		),
		goldmark.WithParserOptions(
			append(
				p.ParserOptions,
				goldmark_parser.WithAttribute(),
				goldmark_parser.WithHeadingAttribute(),
			)...,
		),
		goldmark.WithRendererOptions(
			append(
				p.RenderOptions,
				html.WithUnsafe(),
			)...,
		),
	)
}

func (p MarkdownTransformer) Transform(asset *Asset) error {
	if path.Ext(asset.Path) != ".md" {
		return nil
	}

	html := &bytes.Buffer{}
	if err := p.newGoldmark().Convert(asset.Data, html); err != nil {
		return err
	}

	asset.Path = strings.TrimSuffix(asset.Path, ".md") + ".html"
	asset.Data = html.Bytes()

	return nil
}

func codeWrapperRenderer(w util.BufWriter, context highlighting.CodeBlockContext, entering bool) {
	language, ok := context.Language()
	lang := string(language)

	// code block with a language
	noLang := !ok || language == nil
	if entering {
		w.WriteString(`<figure class="codeblock"`)
		if !noLang {
			w.WriteString(` data-lang="` + lang + `"`)
		}
		w.WriteString(`>`)

		w.WriteString(`<figcaption>`)
		w.WriteString(`<button class="copycode" disabled>Copy</button>`)
		w.WriteString(`</figcaption>`)

		if noLang {
			w.WriteString(`<pre class="chroma">`)
			w.WriteString(`<code>`)
		}
	} else {
		if noLang {
			w.WriteString(`</code>`)
			w.WriteString(`</pre>`)
		}

		w.WriteString(`</figure>`)
	}
}
