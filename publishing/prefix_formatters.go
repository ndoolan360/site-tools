package publishing

// FormatJSComment wraps a notice in a js comment without a trailing newline.
func FormatJSComment(notice string) string {
	return "/* " + notice + " */"
}

// FormatCSSComment wraps a notice in a css comment without a trailing newline.
func FormatCSSComment(notice string) string {
	return "/* " + notice + " */"
}

// FormatHTMLComment wraps a notice in an html comment without a trailing newline.
func FormatHTMLComment(notice string) string {
	// A comment before the DOCTYPE is valid and does not trigger quirks mode.
	return "<!-- " + notice + " -->"
}
