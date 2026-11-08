package images

import (
	"bytes"
	"strings"
	"testing"

	sitetools "go.doolan.dev/site-tools"
)

// TestInlineSVGTransformer covers inlining, accessibility attributes, and
// defense-in-depth filtering of trusted SVG markup.
func TestInlineSVGTransformer(t *testing.T) {
	const iconPath = "/static/images/menu.svg"
	icon := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
  <style>
    .ic { stroke: #111111; }
    @media (prefers-color-scheme: dark) {
      .ic { stroke: #f0f0f0; }
    }
  </style>
  <g class="ic"><line x1="4" y1="6" x2="20" y2="6" /></g>
</svg>`)

	tf := InlineSVGTransformer{SVGs: map[string][]byte{iconPath: icon}}

	t.Run("inlines matching img and strips style", func(t *testing.T) {
		asset := &sitetools.Asset{
			Path: "/index.html",
			Data: []byte(`<a><img class="logo" src="` + iconPath + `" alt="Menu" /></a>`),
		}

		if err := tf.Transform(asset); err != nil {
			t.Fatalf("Transform returned error: %v", err)
		}

		got := string(asset.Data)
		if strings.Contains(got, "<style") {
			t.Errorf("inlined output still contains a <style> block:\n%s", got)
		}
		if strings.Contains(got, "<img") {
			t.Errorf("matching <img> was not replaced:\n%s", got)
		}
		for _, want := range []string{`<svg`, `class="logo"`, `role="img"`, `aria-label="Menu"`, `<line`} {
			if !strings.Contains(got, want) {
				t.Errorf("inlined output missing %q:\n%s", want, got)
			}
		}
	})

	t.Run("strips script, foreignObject, and event handler attributes", func(t *testing.T) {
		maliciousPath := "/static/images/malicious.svg"
		malicious := []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)" viewBox="0 0 24 24">
  <script>alert(2)</script>
  <script src="evil.js" />
  <foreignObject width="1" height="1"><body xmlns="http://www.w3.org/1999/xhtml" onload="alert(3)"><script>alert(4)</script></body></foreignObject>
  <foreignObject width="1" height="1" />
  <g class="ic" onclick='alert(5)'><line x1="4" y1="6" x2="20" y2="6" /></g>
</svg>`)
		tf := InlineSVGTransformer{SVGs: map[string][]byte{maliciousPath: malicious}}
		asset := &sitetools.Asset{
			Path: "/index.html",
			Data: []byte(`<img class="logo" src="` + maliciousPath + `" alt="Menu" />`),
		}

		if err := tf.Transform(asset); err != nil {
			t.Fatalf("Transform returned error: %v", err)
		}

		got := string(asset.Data)
		for _, unwanted := range []string{"<script", "</script>", "<foreignObject", "onload=", "onclick=", "alert("} {
			if strings.Contains(got, unwanted) {
				t.Errorf("inlined output still contains %q:\n%s", unwanted, got)
			}
		}
		if !strings.Contains(got, "<line") {
			t.Errorf("inlined output dropped safe content:\n%s", got)
		}
	})

	t.Run("leaves unmatched img untouched", func(t *testing.T) {
		body := `<img src="/static/images/photo.jpg" alt="Photo" />`
		asset := &sitetools.Asset{Path: "/index.html", Data: []byte(body)}

		if err := tf.Transform(asset); err != nil {
			t.Fatalf("Transform returned error: %v", err)
		}

		if got := string(asset.Data); got != body {
			t.Errorf("Transform mutated unmatched <img>: got %q, want %q", got, body)
		}
	})

	t.Run("decorative images are hidden and not focusable", func(t *testing.T) {
		for name, alt := range map[string]string{"empty alt": ` alt=""`, "missing alt": ""} {
			t.Run(name, func(t *testing.T) {
				asset := &sitetools.Asset{
					Path: "/index.html",
					Data: []byte(`<img src="` + iconPath + `"` + alt + ` />`),
				}

				if err := tf.Transform(asset); err != nil {
					t.Fatalf("Transform returned error: %v", err)
				}

				got := string(asset.Data)
				for _, want := range []string{`<svg`, `aria-hidden="true"`, `focusable="false"`} {
					if !strings.Contains(got, want) {
						t.Errorf("decorative SVG missing %q:\n%s", want, got)
					}
				}
				for _, unwanted := range []string{`<img`, `role="img"`, `aria-label=`} {
					if strings.Contains(got, unwanted) {
						t.Errorf("decorative SVG contains %q:\n%s", unwanted, got)
					}
				}
			})
		}
	})

	t.Run("nil empty and unmatched SVG maps leave HTML untouched", func(t *testing.T) {
		const body = `<p><img src="/static/images/unknown.svg" alt="Unknown" /><img src="/static/images/photo.jpg" alt="Photo" /></p>`
		for name, svgs := range map[string]map[string][]byte{
			"nil":       nil,
			"empty":     {},
			"unmatched": {iconPath: icon},
		} {
			t.Run(name, func(t *testing.T) {
				tf := InlineSVGTransformer{SVGs: svgs}
				asset := &sitetools.Asset{Path: "/index.html", Data: []byte(body)}

				if err := tf.Transform(asset); err != nil {
					t.Fatalf("Transform returned error: %v", err)
				}
				if got := string(asset.Data); got != body {
					t.Errorf("Transform mutated unmatched HTML: got %q, want %q", got, body)
				}
			})
		}
	})

	t.Run("multiple known and unknown images preserve source SVG bytes", func(t *testing.T) {
		const otherPath = "/static/images/other.svg"
		other := []byte(`<svg><circle /></svg>`)
		tf := InlineSVGTransformer{SVGs: map[string][]byte{iconPath: icon, otherPath: other}}
		originals := map[string][]byte{iconPath: bytes.Clone(icon), otherPath: bytes.Clone(other)}
		const unknown = `<img src="/static/images/unknown.svg" alt="Unknown" />`
		const photo = `<img src="/static/images/photo.jpg" alt="Photo" />`
		asset := &sitetools.Asset{
			Path: "/index.html",
			Data: []byte(`<main><img src="` + iconPath + `" class="first" alt="Menu" />` + unknown +
				`<img src="` + otherPath + `" class="second" alt="Other" />` + photo +
				`<img src="` + iconPath + `" class="third" alt="" /></main>`),
		}

		if err := tf.Transform(asset); err != nil {
			t.Fatalf("Transform returned error: %v", err)
		}

		got := string(asset.Data)
		if strings.Count(got, "<svg") != 3 || strings.Count(got, "<img") != 2 {
			t.Errorf("unexpected number of SVGs or remaining images:\n%s", got)
		}
		for _, want := range []string{
			unknown, photo,
			`<svg class="first" role="img" aria-label="Menu" focusable="false"`,
			`<svg class="second" role="img" aria-label="Other" focusable="false"`,
			`<svg class="third" aria-hidden="true" focusable="false"`,
		} {
			if !strings.Contains(got, want) {
				t.Errorf("inlined output missing %q:\n%s", want, got)
			}
		}
		if strings.Contains(got, "<style") {
			t.Errorf("inlined output still contains a <style> block:\n%s", got)
		}
		for path, original := range originals {
			if !bytes.Equal(tf.SVGs[path], original) {
				t.Errorf("Transform mutated source SVG bytes for %q", path)
			}
		}
	})

	t.Run("ignores unsupported source attribute quoting", func(t *testing.T) {
		for _, src := range []string{`src='` + iconPath + `'`, `src=` + iconPath} {
			body := `<img ` + src + ` alt="Menu" />`
			asset := &sitetools.Asset{Path: "/index.html", Data: []byte(body)}

			if err := tf.Transform(asset); err != nil {
				t.Fatalf("Transform returned error: %v", err)
			}
			if got := string(asset.Data); got != body {
				t.Errorf("Transform mutated unsupported HTML: got %q, want %q", got, body)
			}
		}
	})

	t.Run("ignores non-html assets", func(t *testing.T) {
		asset := &sitetools.Asset{Path: iconPath, Data: append([]byte(nil), icon...)}

		if err := tf.Transform(asset); err != nil {
			t.Fatalf("Transform returned error: %v", err)
		}

		if got := string(asset.Data); got != string(icon) {
			t.Errorf("Transform mutated a non-HTML asset:\n%s", got)
		}
	})
}
