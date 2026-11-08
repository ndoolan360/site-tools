package publishing

import (
	"path"
	"strings"

	sitetools "go.doolan.dev/site-tools"
)

var defaultPrefixFormatters = map[string]func(string) string{
	".js":   FormatJSComment,
	".css":  FormatCSSComment,
	".html": FormatHTMLComment,
}

// PrefixCommentTransformer prepends its Notice as a leading comment to files.
// by default. Register adds or overrides formatting for file extensions.
type PrefixCommentTransformer struct {
	Notice string

	formatters map[string]func(string) string
}

var _ sitetools.Transformer = PrefixCommentTransformer{}

// Register sets a comment formatter for an extension, with or without a leading
// dot (for example, ".py" or "py"). Matching is case-sensitive; an empty extension
// matches extensionless files. Registrations override formatters.
// (built-in formatters include JS, CSS, HTML) and any previous registration for the same extension.
func (t *PrefixCommentTransformer) Register(extension string, formatter func(string) string) {
	if extension != "" && !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}
	if formatter == nil {
		delete(t.formatters, extension)
		return
	}
	if t.formatters == nil {
		t.formatters = make(map[string]func(string) string)
	}
	t.formatters[extension] = formatter
}

func (t PrefixCommentTransformer) Transform(asset *sitetools.Asset) error {
	extension := path.Ext(asset.Path)
	formatter := t.formatters[extension]
	if formatter == nil {
		formatter = defaultPrefixFormatters[extension]
	}
	if formatter == nil {
		return nil
	}

	asset.Data = append([]byte(formatter(t.Notice)+"\n"), asset.Data...)
	return nil
}
