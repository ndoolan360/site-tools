package builder

import (
	"bytes"
	"testing"
)

func TestAddAutoReload_Transform(t *testing.T) {
	tests := []struct {
		name         string
		asset        *Asset
		autoReload   *AddAutoReload
		expectedData []byte
		expectError  bool
	}{
		{
			name: "HTML file",
			asset: &Asset{
				Path: "index.html",
				Data: []byte("<html><body></body></html>"),
			},
			autoReload: &AddAutoReload{
				WebSocketPath: "/ws",
				Timeout:       1000,
			},
			expectedData: []byte("<html><body><script>new WebSocket(\"ws://\"+location.host+\"/ws\").onclose=()=>setTimeout(()=>location.reload(!0),1000)</script></body></html>"),
			expectError:  false,
		},
		{
			name: "Non-HTML file",
			asset: &Asset{
				Path: "style.css",
				Data: []byte("body { color: red; }"),
			},
			autoReload: &AddAutoReload{
				WebSocketPath: "/ws",
				Timeout:       1000,
			},
			expectedData: []byte("body { color: red; }"),
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.autoReload.Transform(tt.asset)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !bytes.Equal(tt.asset.Data, tt.expectedData) {
					t.Errorf("expected data %q, got %q", tt.expectedData, tt.asset.Data)
				}
			}
		})
	}
}
