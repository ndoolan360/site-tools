package publishing

import (
	"fmt"
	"strings"
	"testing"

	sitetools "go.doolan.dev/site-tools"
	"go.doolan.dev/site-tools/minify"
)

// TestPrefixCommentTransformer pins down the per-file-type comment
// delimiters, the leading placement, and the fact that unsupported asset types
// are passed through untouched. See PrefixCommentTransformer in prefix.go.
func TestPrefixCommentTransformer(t *testing.T) {
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

			if err := (PrefixCommentTransformer{Notice: notice}).Transform(asset); err != nil {
				t.Fatalf("Transform(%s) returned error: %v", tc.path, err)
			}

			if got := string(asset.Data); got != tc.want {
				t.Errorf("Transform(%s) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

func TestPrefixCommentFormatters(t *testing.T) {
	for _, tc := range []struct {
		name      string
		formatter func(string) string
		want      string
	}{
		{"JavaScript", FormatJSComment, "/* NOTICE */"},
		{"CSS", FormatCSSComment, "/* NOTICE */"},
		{"HTML", FormatHTMLComment, "<!-- NOTICE -->"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.formatter("NOTICE"); got != tc.want {
				t.Errorf("formatter = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPrefixCommentTransformer_RegisterDefaultFormatters(t *testing.T) {
	for _, tc := range []struct {
		extension string
		formatter func(string) string
		want      string
	}{
		{".mjs", FormatJSComment, "/* NOTICE */\nBODY"},
		{".scss", FormatCSSComment, "/* NOTICE */\nBODY"},
		{".htm", FormatHTMLComment, "<!-- NOTICE -->\nBODY"},
	} {
		t.Run(tc.extension, func(t *testing.T) {
			transformer := PrefixCommentTransformer{Notice: "NOTICE"}
			transformer.Register(tc.extension, tc.formatter)
			asset := &sitetools.Asset{Path: "/file" + tc.extension, Data: []byte("BODY")}
			if err := transformer.Transform(asset); err != nil {
				t.Fatal(err)
			}
			if got := string(asset.Data); got != tc.want {
				t.Errorf("Transform(%s) = %q, want %q", asset.Path, got, tc.want)
			}
		})
	}
}

func TestPrefixCommentTransformer_Register(t *testing.T) {
	for _, tc := range []struct {
		name      string
		extension string
		path      string
	}{
		{"extension with dot", ".py", "/scripts/task.py"},
		{"extension without dot", "yaml", "/config.yaml"},
		{"extensionless file", "", "/LICENSE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			transformer := PrefixCommentTransformer{Notice: "NOTICE"}
			calls := 0
			transformer.Register(tc.extension, func(notice string) string {
				calls++
				if notice != "NOTICE" {
					t.Errorf("formatter received %q, want NOTICE", notice)
				}
				return "# " + notice
			})
			asset := &sitetools.Asset{Path: tc.path, Data: []byte("BODY"), Meta: map[string]any{"Title": "Title"}}
			if err := (sitetools.Assets{asset}).Transform(transformer); err != nil {
				t.Fatal(err)
			}
			if got := string(asset.Data); got != "# NOTICE\nBODY" {
				t.Errorf("registered format = %q, want %q", got, "# NOTICE\nBODY")
			}
			if calls != 1 {
				t.Errorf("formatter called %d times, want 1", calls)
			}
			if asset.Path != tc.path || asset.Meta["Title"] != "Title" {
				t.Errorf("transform changed asset path or metadata: %+v", asset)
			}
		})
	}
}

func TestPrefixCommentTransformer_RegisterOverridesAndIsLocal(t *testing.T) {
	custom := PrefixCommentTransformer{Notice: "NOTICE"}
	custom.Register(".js", func(notice string) string { return "// " + notice })
	custom.Register(".py", func(notice string) string { return "# " + notice })

	for _, tc := range []struct {
		name        string
		transformer PrefixCommentTransformer
		path        string
		want        string
	}{
		{"override built-in", custom, "/script.js", "// NOTICE\nBODY"},
		{"preserve other built-ins", custom, "/style.css", "/* NOTICE */\nBODY"},
		{"preserve other registrations", custom, "/script.py", "# NOTICE\nBODY"},
		{"leave unknown extensions", custom, "/data.json", "BODY"},
		{"matching is case-sensitive", custom, "/script.PY", "BODY"},
		{"other instance retains built-in", PrefixCommentTransformer{Notice: "NOTICE"}, "/script.js", "/* NOTICE */\nBODY"},
		{"other instance lacks registration", PrefixCommentTransformer{Notice: "NOTICE"}, "/script.py", "BODY"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			asset := &sitetools.Asset{Path: tc.path, Data: []byte("BODY")}
			if err := tc.transformer.Transform(asset); err != nil {
				t.Fatal(err)
			}
			if got := string(asset.Data); got != tc.want {
				t.Errorf("Transform(%s) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

func TestPrefixCommentTransformer_RegisterReplacesAndRemoves(t *testing.T) {
	transformer := PrefixCommentTransformer{}
	transformer.Register(".unused", nil)
	transformer.Register(".py", func(notice string) string { return "old " + notice })
	transformer.Register("py", func(notice string) string { return "# " + notice })
	transformer.Notice = "UPDATED"

	asset := &sitetools.Asset{Path: "/script.py", Data: []byte("BODY")}
	if err := transformer.Transform(asset); err != nil {
		t.Fatal(err)
	}
	if got := string(asset.Data); got != "# UPDATED\nBODY" {
		t.Errorf("replacement formatter = %q, want %q", got, "# UPDATED\nBODY")
	}

	transformer.Register(".js", func(notice string) string { return "// " + notice })
	transformer.Register("js", nil)
	transformer.Register("py", nil)
	for _, tc := range []struct {
		path string
		want string
	}{
		{"/script.js", "/* UPDATED */\nBODY"},
		{"/script.py", "BODY"},
	} {
		asset := &sitetools.Asset{Path: tc.path, Data: []byte("BODY")}
		if err := transformer.Transform(asset); err != nil {
			t.Fatal(err)
		}
		if got := string(asset.Data); got != tc.want {
			t.Errorf("Transform(%s) after removing registration = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func ExamplePrefixCommentTransformer_Register() {
	transformer := PrefixCommentTransformer{Notice: "Generated file"}
	transformer.Register(".py", func(notice string) string {
		return "# " + notice
	})

	asset := &sitetools.Asset{Path: "/task.py", Data: []byte("print('hello')\n")}
	if err := transformer.Transform(asset); err != nil {
		panic(err)
	}
	fmt.Print(string(asset.Data))
	// Output:
	// # Generated file
	// print('hello')
}

func TestPrefixCommentTransformer_AfterMinification(t *testing.T) {
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

			if err := (PrefixCommentTransformer{Notice: "NOTICE"}).Transform(asset); err != nil {
				t.Fatalf("Transform(%s) returned error: %v", tc.path, err)
			}
			if got, want := string(asset.Data), tc.comment+"\n"+minified; got != want {
				t.Errorf("Transform(%s) = %q, want %q", tc.path, got, want)
			}
		})
	}
}
