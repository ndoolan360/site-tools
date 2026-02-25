package sitetools

import (
	"bytes"
	_ "embed"
	"fmt"
	"path"
)

//go:embed assets/autoreload.js
var autoreloadTemplate []byte

// AddAutoReload is a transformer that injects a script into HTML files to enable
// auto-reloading when a WebSocket connection is closed.
// This is useful for development environments where you want the page to automatically
// refresh when the server restarts or files change. Not really intended for production use.
type AddAutoReload struct {
	WebSocketPath string
	Timeout       int
}

func (auto AddAutoReload) Transform(asset *Asset) error {
	if path.Ext(asset.Path) != ".html" {
		return nil
	}

	end := bytes.Index(asset.Data, []byte("</body>"))
	if end == -1 {
		return nil
	}

	scriptAsset := &Asset{
		Path: "autoreload.js",
		Data: autoreloadTemplate,
	}

	// Use ReplacerTransformer to replace placeholders
	err := ReplacerTransformer{
		Replacements: map[string]string{
			"{{.WEBSOCKET_PATH}}": auto.WebSocketPath,
			"{{.TIMEOUT}}":        fmt.Sprintf("%d", auto.Timeout),
		},
	}.Transform(scriptAsset)
	if err != nil {
		return err
	}

	wrappedScript := "<script>" + string(scriptAsset.Data) + "</script>"
	asset.Data = append(asset.Data[:end], append([]byte(wrappedScript), asset.Data[end:]...)...)

	return nil
}
