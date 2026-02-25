package sitetools

import (
	"testing"
)

func TestTemplate_WithComponents(t *testing.T) {
	asset := &Asset{
		Path: "/page.html",
		Data: []byte(`{{ template "comp1" . }}<p>{{ .PageContent }}</p>`),
		Meta: map[string]any{"PageContent": "Page specific stuff"},
	}

	component1 := &Asset{
		Path: "/components/comp1.html",
		Data: []byte(`<comp1>Component 1: {{ .Global.Comp1Data }}</comp1>`),
	}

	err := TemplateTransformer{
		Components: map[string]*Asset{"comp1": component1},
		Global:     map[string]any{"Comp1Data": "Global for Comp1"},
	}.Transform(asset)
	if err != nil {
		t.Fatalf("Transform returned an unexpected error: %v", err)
	}

	expectedData := `<comp1>Component 1: Global for Comp1</comp1><p>Page specific stuff</p>`
	if string(asset.Data) != expectedData {
		t.Errorf("Transform with components did not produce the expected output.\nExpected:\n%s\nGot:\n%s", expectedData, string(asset.Data))
	}
}

func TestTemplate_WithNilMeta(t *testing.T) {
	asset := &Asset{
		Path: "/page.html",
		Data: []byte(`{{ .Global.SiteName }} - {{ .PageContent }}`),
		Meta: nil,
	}

	transformer := TemplateTransformer{
		Global: map[string]any{"SiteName": "MySite"},
	}

	err := transformer.Transform(asset)
	if err != nil {
		t.Fatalf("Transform returned an unexpected error: %v", err)
	}

	expectedData := "MySite - <no value>"
	if string(asset.Data) != expectedData {
		t.Errorf("Transform with nil meta did not produce the expected output.\nExpected:\n%s\nGot:\n%s", expectedData, string(asset.Data))
	}
}

func TestTemplate_WithMalformedComponent(t *testing.T) {
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

func TestTemplate_WithGlobalInMeta(t *testing.T) {
	asset := &Asset{
		Path: "/page.html",
		Data: []byte(`{{ .Global.SiteName }} - {{ .PageContent }}`),
		Meta: map[string]any{
			"PageContent": "Page specific stuff",
			"Global":      "This should cause an error", // Reserved key
		},
	}

	transformer := TemplateTransformer{
		Global: map[string]any{"SiteName": "MySite"},
	}

	err := transformer.Transform(asset)
	if err == nil {
		t.Fatal("Transform expected an error due to reserved 'Global' key in asset meta, but got nil")
	}
}
