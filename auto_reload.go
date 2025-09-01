package sitetools

import (
	"bytes"
	"fmt"
	"path"
)

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

	scriptFmt := `<script>new WebSocket("ws://"+location.host+"%s").onclose=()=>setTimeout(()=>location.reload(!0),%d)</script>`
	script := []byte(fmt.Sprintf(scriptFmt, auto.WebSocketPath, auto.Timeout))

	asset.Data = append(asset.Data[:end], append(script, asset.Data[end:]...)...)

	return nil
}
