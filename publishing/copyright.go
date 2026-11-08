package publishing

import (
	"path"

	sitetools "go.doolan.dev/site-tools"
)

// CopyrightCommentTransformer prepends its Notice as a leading comment to JS,
// CSS, and HTML assets.
//
// It must run after minification: minifiers strip ordinary comments, so
// stamping earlier would discard the notice. Running last also guarantees the
// notice sits at the very top of the file that actually ships. Comment
// delimiters are chosen per file type; any other asset is left untouched.
type CopyrightCommentTransformer struct {
	Notice string
}

var _ sitetools.Transformer = CopyrightCommentTransformer{}

func (t CopyrightCommentTransformer) Transform(asset *sitetools.Asset) error {
	var comment string
	switch path.Ext(asset.Path) {
	case ".js", ".css":
		comment = "/* " + t.Notice + " */"
	case ".html":
		// A comment before the DOCTYPE is valid and does not trigger quirks
		// mode, so the notice can safely lead the document.
		comment = "<!-- " + t.Notice + " -->"
	default:
		return nil
	}

	asset.Data = append([]byte(comment+"\n"), asset.Data...)
	return nil
}
