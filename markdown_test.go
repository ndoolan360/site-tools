package sitetools

import (
	"bufio"
	"bytes"
	"testing"

	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

func TestMarkdownTransformer_TransformMarkdown(t *testing.T) {
	transformer := MarkdownTransformer{}

	asset := &Asset{
		Path: "test.md",
		Data: []byte("# Hello"),
	}
	err := transformer.Transform(asset)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}
	if asset.Path != "test.html" {
		t.Errorf("Expected path to be 'test.html', got '%s'", asset.Path)
	}

	expectedHTML := "<h1>Hello</h1>\n"
	if string(asset.Data) != expectedHTML {
		t.Errorf("Expected HTML to be %q, got %q", expectedHTML, string(asset.Data))
	}
}

func TestMarkdownTransformer_IgnoresNonMarkdown(t *testing.T) {
	transformer := MarkdownTransformer{}

	asset := &Asset{
		Path: "test.txt",
		Data: []byte("Just some text."),
	}
	originalData := make([]byte, len(asset.Data))
	copy(originalData, asset.Data)
	originalPath := asset.Path

	err := transformer.Transform(asset)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}
	if asset.Path != originalPath {
		t.Errorf("Expected path to be '%s', got '%s'", originalPath, asset.Path)
	}
	if !bytes.Equal(asset.Data, originalData) {
		t.Errorf("Expected data to be unchanged, got '%s'", string(asset.Data))
	}
}

type MockCodeBlockContext struct {
	highlighting.CodeBlockContext
	lang []byte
}

func (m *MockCodeBlockContext) Language() ([]byte, bool) {
	if m.lang == nil {
		return nil, false
	}
	return m.lang, true
}
func (m *MockCodeBlockContext) Highlighted() bool                            { return false }
func (m *MockCodeBlockContext) Attributes() highlighting.ImmutableAttributes { return nil }

func TestCodeWrapperRenderer(t *testing.T) {
	tests := []struct {
		name     string
		context  *MockCodeBlockContext
		entering bool
		expected string
	}{
		{
			name: "Entering with language",
			context: &MockCodeBlockContext{
				lang: []byte("go"),
			},
			entering: true,
			expected: `<figure class="codeblock" data-lang="go"><figcaption><button class="copycode" disabled>Copy</button></figcaption>`,
		},
		{
			name: "Exiting with language",
			context: &MockCodeBlockContext{
				lang: []byte("go"),
			},
			entering: false,
			expected: `</figure>`,
		},
		{
			name: "Entering without language",
			context: &MockCodeBlockContext{
				lang: nil,
			},
			entering: true,
			expected: `<figure class="codeblock"><figcaption><button class="copycode" disabled>Copy</button></figcaption><pre class="chroma"><code>`,
		},
		{
			name: "Exiting without language",
			context: &MockCodeBlockContext{
				lang: nil,
			},
			entering: false,
			expected: `</code></pre></figure>`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer := bufio.NewWriter(&buf)
			codeWrapperRenderer(writer, test.context, test.entering)
			writer.Flush()

			if buf.String() != test.expected {
				t.Errorf("Expected %q, got %q", test.expected, buf.String())
			}
		})
	}
}
