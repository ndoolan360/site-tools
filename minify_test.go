package builder

import "testing"

func TestMinifyTransformer_Transform(t *testing.T) {
	transformer := &MinifyTransformer{}
	testCases := []struct {
		name      string
		extension string
		input     string
		expected  string
	}{
		{
			name:      "HTML",
			extension: "html",
			input:     "<html>  <body>   <h1> Hello World </h1> </body></html>",
			expected:  "<h1>Hello World</h1>",
		},
		{
			name:      "CSS",
			extension: "css",
			input:     "body {  margin: 0;   padding: 0; }",
			expected:  "body{margin:0;padding:0}",
		},
		{
			name:      "JavaScript",
			extension: "js",
			input:     `function test() {  console.log("Hello World"); }`,
			expected:  `function test(){console.log("Hello World")}`,
		},
		{
			name:      "SVG",
			extension: "svg",
			input:     `<svg>  <circle cx="50" cy="50" r="40" /></svg>`,
			expected:  `<svg><circle cx="50" cy="50" r="40"/></svg>`,
		},
		{
			name:      "Plain Text",
			extension: "txt",
			input:     "This is a plain text file.",
			expected:  "This is a plain text file.",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			asset := &Asset{
				Path: "/test." + tc.extension,
				Data: []byte(tc.input),
			}
			err := transformer.Transform(asset)
			if err != nil {
				t.Fatalf("Transform() error = %v", err)
			}
			if string(asset.Data) != tc.expected {
				t.Errorf("Transform() = %v, want %v", string(asset.Data), tc.expected)
			}
		})
	}
}
