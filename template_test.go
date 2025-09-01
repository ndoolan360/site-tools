package sitetools

import (
	"testing"
)

func TestTemplate_Transform_WithComponents(t *testing.T) {

	asset := &Asset{
		Path: "/page.html",
		Data: []byte(`{{ template "comp1" . }}<p>{{ .PageContent }}</p>`),
		Meta: map[string]any{
			"PageContent": "Page specific stuff",
			"Title":       "Wrapped Page with Component",
		},
	}

	component1 := &Asset{
		Path: "/components/comp1.html",
		Data: []byte(`<comp1>Component 1: {{ .Global.Comp1Data }}</comp1>`),
	}

	err := TemplateTransformer{
		Components: map[string]*Asset{"comp1": component1},
		GlobalData: map[string]any{"Comp1Data": "Global for Comp1"},
	}.Transform(asset)
	if err != nil {
		t.Fatalf("Transform returned an unexpected error: %v", err)
	}

	expectedData := `<comp1>Component 1: Global for Comp1</comp1><p>Page specific stuff</p>`
	if string(asset.Data) != expectedData {
		t.Errorf("Transform with components did not produce the expected output.\nExpected:\n%s\nGot:\n%s", expectedData, string(asset.Data))
	}
}

func TestTemplate_Transform_WithMalformedComponent(t *testing.T) {
	asset := &Asset{
		Path: "/page.html",
		Data: []byte(`{{ template "bad_comp" . }}`),
		Meta: map[string]any{},
	}
	badComponent := &Asset{
		Path: "/components/bad.html",
		Data: []byte("{{ .Global.Unclosed "), // Malformed
		Meta: map[string]any{},
	}
	transformer := TemplateTransformer{
		Components: map[string]*Asset{
			"bad_comp": badComponent,
		},
	}
	err := transformer.Transform(asset)
	if err == nil {
		t.Fatal("Transform expected an error due to malformed component template, but got nil")
	}
}

func TestTemplate_TransformWithWrapper_WithComponents(t *testing.T) {
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
		Data: []byte(`<html><head><title>{{ .Asset.Title }}</title></head><body>{{ template "content" . }}<footer>{{ template "comp2" . }}</footer></body></html>`),
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

	transformer := TemplateTransformer{
		GlobalData: map[string]any{
			"Comp1Data": "Global for Comp1",
			"Comp2Data": "Global for Comp2",
		},
		Components: map[string]*Asset{
			"comp1":          component1,
			"comp2":          component2,
			"unused-md-comp": {Path: "/components/unused.md", Data: []byte("Unused component")},
		},
	}

	err := transformer.TransformWithWrapper(asset, WrapperTemplate{
		Template:       wrapperAsset,
		ChildBlockName: "content",
	})
	if err != nil {
		t.Fatalf("Transform returned an unexpected error: %v", err)
	}

	expectedData := `<html><head><title>Wrapped Page with Component</title></head><body><comp1>Component 1: Global for Comp1</comp1><p>Page specific stuff</p><footer><comp2>Component 2: Global for Comp2</comp2></footer></body></html>`
	if string(asset.Data) != expectedData {
		t.Errorf("Transform with wrapper and components did not produce the expected output.\nExpected:\n%s\nGot:\n%s", expectedData, string(asset.Data))
	}
}

func TestTemplate_TransformWithWrapper_MalformedWrapper(t *testing.T) {
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
	transformer := TemplateTransformer{}
	err := transformer.TransformWithWrapper(asset, WrapperTemplate{
		Template:       wrapperAsset,
		ChildBlockName: "content",
	})
	if err == nil {
		t.Fatal("Transform expected an error due to malformed wrapper template, but got nil")
	}
}
