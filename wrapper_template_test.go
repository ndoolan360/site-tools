package sitetools

import "testing"

func TestWrapperTemplate_WithComponents(t *testing.T) {
	asset := &Asset{
		Path: "/page.html",
		Data: []byte(`{{ template "comp1" . }}<p>{{ .PageContent }}</p>`),
		Meta: map[string]any{
			"PageContent": "Page specific stuff",
			"Title":       "Wrapped Page with Component",
		},
	}

	wrapperAsset := &Asset{
		Path: "/wrapper.html",
		Data: []byte(`<html><head><title>{{ .Title }}</title></head><body>{{ template "content" . }}<footer>{{ template "comp2" . }}</footer></body></html>`),
		Meta: map[string]any{},
	}

	component1 := &Asset{
		Path: "/components/comp1.html",
		Data: []byte(`<comp1>Component 1: {{ .Global.Comp1Data }}</comp1>`),
		Meta: map[string]any{},
	}
	component2 := &Asset{
		Path: "/components/comp2.html",
		Data: []byte(`<comp2>Component 2: {{ .Global.Comp2Data }}</comp2>`),
		Meta: map[string]any{},
	}

	transformer := WrapperTemplateTransformer{
		TemplateTransformer: TemplateTransformer{
			Global: map[string]any{
				"Comp1Data": "Global for Comp1",
				"Comp2Data": "Global for Comp2",
			},
			Components: map[string]*Asset{
				"comp1":          component1,
				"comp2":          component2,
				"unused-md-comp": {Path: "/components/unused.md", Data: []byte("Unused component")},
			},
		},
		WrapperTemplate: WrapperTemplate{
			Template:       wrapperAsset,
			ChildBlockName: "content",
		},
	}

	err := transformer.Transform(asset)
	if err != nil {
		t.Fatalf("Transform returned an unexpected error: %v", err)
	}

	expectedData := `<html><head><title>Wrapped Page with Component</title></head><body><comp1>Component 1: Global for Comp1</comp1><p>Page specific stuff</p><footer><comp2>Component 2: Global for Comp2</comp2></footer></body></html>`
	if string(asset.Data) != expectedData {
		t.Errorf("Transform with wrapper and components did not produce the expected output.\nExpected:\n%s\nGot:\n%s", expectedData, string(asset.Data))
	}
}

func TestWrapperTemplate_MalformedWrapper(t *testing.T) {
	asset := &Asset{
		Path: "/page.html",
		Data: []byte("Page content"),
		Meta: map[string]any{},
	}
	wrapperAsset := &Asset{
		Path: "/wrapper.html",
		Data: []byte("{{ define \"content\" }} {{ end }} {{ .Global.Unclosed "), // Malformed
		Meta: map[string]any{},
	}
	transformer := WrapperTemplateTransformer{
		WrapperTemplate: WrapperTemplate{
			Template:       wrapperAsset,
			ChildBlockName: "content",
		},
	}
	err := transformer.Transform(asset)
	if err == nil {
		t.Fatal("Transform expected an error due to malformed wrapper template, but got nil")
	}
}

func TestWrapperTemplate_MissingTemplate(t *testing.T) {
	asset := &Asset{
		Path: "/page.html",
		Data: []byte("Page content"),
		Meta: map[string]any{},
	}

	transformer := WrapperTemplateTransformer{
		WrapperTemplate: WrapperTemplate{
			Template:       nil,
			ChildBlockName: "content",
		},
	}

	err := transformer.Transform(asset)
	if err == nil {
		t.Fatal("Transform expected an error due to missing wrapper template, but got nil")
	}
}
