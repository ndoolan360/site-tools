package publishing

import (
	"strings"
	"testing"

	sitetools "go.doolan.dev/site-tools"
	"go.doolan.dev/site-tools/minify"
)

// TestCopyrightCommentTransformer pins down the per-file-type comment
// delimiters, the leading placement, and the fact that unsupported asset types
// are passed through untouched. See CopyrightCommentTransformer in
// copyright.go.
func TestCopyrightCommentTransformer(t *testing.T) {
	const notice = "NOTICE"
	const body = "BODY"

	cases := []struct {
		name string
		path string
		want string
	}{
		{"js gets a block comment", "/static/scripts/game.js", "/* NOTICE */\nBODY"},
		{"css gets a block comment", "/static/styles/style.css", "/* NOTICE */\nBODY"},
		{"html gets an html comment", "/index.html", "<!-- NOTICE -->\nBODY"},
		{"json is left untouched", "/static/data/meta.json", "BODY"},
		{"svg is left untouched", "/static/favicon.svg", "BODY"},
		{"extensionless asset is left untouched", "/robots", "BODY"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			asset := &sitetools.Asset{Path: tc.path, Data: []byte(body)}

			if err := (CopyrightCommentTransformer{Notice: notice}).Transform(asset); err != nil {
				t.Fatalf("Transform(%s) returned error: %v", tc.path, err)
			}

			if got := string(asset.Data); got != tc.want {
				t.Errorf("Transform(%s) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

func TestCopyrightCommentTransformer_AfterMinification(t *testing.T) {
	cases := []struct {
		path    string
		body    string
		comment string
	}{
		{"/script.js", "/* ordinary comment */\nconst answer = 42;", "/* NOTICE */"},
		{"/style.css", "/* ordinary comment */\nbody { color: red; }", "/* NOTICE */"},
		{"/index.html", "<!-- ordinary comment -->\n<!DOCTYPE html><html><body>hello</body></html>", "<!-- NOTICE -->"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			asset := &sitetools.Asset{Path: tc.path, Data: []byte(tc.body)}
			if err := (minify.MinifyTransformer{}).Transform(asset); err != nil {
				t.Fatalf("Minify(%s) returned error: %v", tc.path, err)
			}
			minified := string(asset.Data)
			if strings.Contains(minified, "ordinary comment") {
				t.Fatalf("Expected minification to remove the ordinary comment, got %q", minified)
			}

			if err := (CopyrightCommentTransformer{Notice: "NOTICE"}).Transform(asset); err != nil {
				t.Fatalf("Transform(%s) returned error: %v", tc.path, err)
			}
			if got, want := string(asset.Data), tc.comment+"\n"+minified; got != want {
				t.Errorf("Transform(%s) = %q, want %q", tc.path, got, want)
			}
		})
	}
}
