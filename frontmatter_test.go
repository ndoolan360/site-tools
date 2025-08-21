package builder

import (
	"bytes"
	"reflect"
	"testing"
)

func TestCollectFrontMatter_Transform(t *testing.T) {
	tests := []struct {
		name         string
		asset        *Asset
		expectedData []byte
		expectedMeta map[string]any
	}{
		{
			name: "Valid frontmatter with pre-initialized meta",
			asset: &Asset{
				Data: []byte("---\ntitle: Hello\ntags: [go, test]\n---\nContent here"),
				Meta: make(map[string]any),
			},
			expectedData: []byte("Content here"),
			expectedMeta: map[string]any{
				"title": "Hello",
				"tags":  []any{"go", "test"},
			},
		},
		{
			name: "Valid frontmatter with nil meta",
			asset: &Asset{
				Data: []byte("---\nkey: value\n---\nSome data"),
			},
			expectedData: []byte("Some data"),
			expectedMeta: map[string]any{
				"key": "value",
			},
		},
		{
			name: "No frontmatter",
			asset: &Asset{
				Data: []byte("Just content here"),
				Meta: make(map[string]any),
			},
			expectedData: []byte("Just content here"),
			expectedMeta: map[string]any{},
		},
		{
			name: "Existing meta gets merged, frontmatter overwrites",
			asset: &Asset{
				Data: []byte("---\ntitle: New Title\nauthor: Gem\n---\nMore content"),
				Meta: map[string]any{
					"existingKey": "existingValue",
					"author":      "Original Author", // This should be overwritten
				},
			},
			expectedData: []byte("More content"),
			expectedMeta: map[string]any{
				"title":       "New Title",
				"author":      "Gem",
				"existingKey": "existingValue",
			},
		},
	}

	transformer := CollectFrontMatter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := transformer.Transform(tt.asset)

			if err != nil {
				t.Fatalf("did not expect an error, but got: %v", err)
			}

			if !bytes.Equal(tt.asset.Data, tt.expectedData) {
				t.Errorf("expected data %q, got %q", string(tt.expectedData), string(tt.asset.Data))
			}

			if tt.asset.Meta == nil && len(tt.expectedMeta) > 0 {
				t.Errorf("asset.Meta is nil, but expected %v", tt.expectedMeta)
			} else if tt.asset.Meta != nil && len(tt.expectedMeta) == 0 && len(tt.asset.Meta) != 0 {
				t.Errorf("expected empty meta, got %v", tt.asset.Meta)
			} else if !reflect.DeepEqual(tt.asset.Meta, tt.expectedMeta) {
				t.Errorf("expected meta %v, got %v", tt.expectedMeta, tt.asset.Meta)
			}
		})
	}
}
