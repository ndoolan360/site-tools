package images

import (
	"bytes"
	"fmt"
	"path"
	"regexp"

	sitetools "go.doolan.dev/site-tools"
)

var (
	imgTagRe    = regexp.MustCompile(`(?is)<img\b[^>]*>`)
	styleTagRe  = regexp.MustCompile(`(?is)\s*<style\b[^>]*>.*?</style>`)
	srcAttrRe   = regexp.MustCompile(`(?i)\bsrc="([^"]*)"`)
	classAttrRe = regexp.MustCompile(`(?i)\bclass="([^"]*)"`)
	altAttrRe   = regexp.MustCompile(`(?i)\balt="([^"]*)"`)

	// Defense-in-depth filtering for trusted SVGs, not a full security sanitizer.
	scriptTagRe        = regexp.MustCompile(`(?is)\s*<script\b[^>]*>.*?</script>`)
	selfCloseTagRe     = regexp.MustCompile(`(?is)<(script|foreignObject)\b[^>]*/>`)
	foreignObjectRe    = regexp.MustCompile(`(?is)\s*<foreignObject\b[^>]*>.*?</foreignObject>`)
	eventHandlerAttrRe = regexp.MustCompile(`(?i)\s+on[a-z]+\s*=\s*("[^"]*"|'[^']*')`)
)

// InlineSVGTransformer replaces matching <img> elements in .html assets with
// trusted first-party SVG markup from SVGs. It preserves the image's class and
// derives accessibility attributes from its alt text. Unmatched images and
// non-HTML assets are left unchanged.
//
// Only HTML <img> attributes with double-quoted values are supported. Run this
// transformer before HTML minification, since minifiers may unquote attributes.
//
// SVG <style> blocks are removed; page CSS is expected to provide the styling.
// Script and foreignObject elements and quoted event handler attributes are
// filtered as defense in depth. This regex filtering is NOT a full security
// sanitizer and must not be used with untrusted SVGs or arbitrary HTML.
type InlineSVGTransformer struct {
	// SVGs maps an asset path (e.g. "/static/images/menu.svg") to trusted SVG
	// bytes. The source bytes are not modified.
	SVGs map[string][]byte
}

var _ sitetools.Transformer = InlineSVGTransformer{}

func (t InlineSVGTransformer) Transform(asset *sitetools.Asset) error {
	if path.Ext(asset.Path) != ".html" {
		return nil
	}

	asset.Data = imgTagRe.ReplaceAllFunc(asset.Data, func(tag []byte) []byte {
		svg, ok := t.SVGs[attrValue(srcAttrRe, tag)]
		if !ok {
			return tag
		}
		return inlineSVG(svg, attrValue(classAttrRe, tag), attrValue(altAttrRe, tag))
	})
	return nil
}

// attrValue returns the first capture group of re within tag, or "" if absent.
func attrValue(re *regexp.Regexp, tag []byte) string {
	if m := re.FindSubmatch(tag); m != nil {
		return string(m[1])
	}
	return ""
}

// inlineSVG injects presentation/accessibility attributes onto the root <svg>:
// the original <img>'s class (so existing sizing rules still match), plus
// either an accessible name (when the <img> carried non-empty alt text) or
// aria-hidden (for decorative icons, mirroring alt="").
//
// Style blocks are stripped so page CSS can style the inline SVG without
// requiring inline styles in the page's content security policy. Script and
// foreignObject elements and quoted event handler attributes are also stripped
// as defense in depth for trusted SVGs, not as a full security sanitizer.
func inlineSVG(svg []byte, class, alt string) []byte {
	svg = styleTagRe.ReplaceAll(svg, nil)
	svg = scriptTagRe.ReplaceAll(svg, nil)
	svg = foreignObjectRe.ReplaceAll(svg, nil)
	svg = selfCloseTagRe.ReplaceAll(svg, nil)
	svg = eventHandlerAttrRe.ReplaceAll(svg, nil)

	var attrs bytes.Buffer
	if class != "" {
		fmt.Fprintf(&attrs, ` class="%s"`, class)
	}
	if alt != "" {
		fmt.Fprintf(&attrs, ` role="img" aria-label="%s"`, alt)
	} else {
		attrs.WriteString(` aria-hidden="true"`)
	}
	attrs.WriteString(` focusable="false"`)

	return bytes.Replace(svg, []byte("<svg"), append([]byte("<svg"), attrs.Bytes()...), 1)
}
