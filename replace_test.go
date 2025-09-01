package sitetools

import (
	"bytes"
	"testing"
)

func TestReplacerTransformer_Transform_BasicReplacement(t *testing.T) {
	asset := &Asset{
		Path: "/test.html",
		Data: []byte("Hello World"),
	}
	transformer := ReplacerTransformer{
		Replacements: map[string]string{
			"World": "Go",
		},
	}

	err := transformer.Transform(asset)
	if err != nil {
		t.Fatalf("Transform returned an error: %v", err)
	}

	expectedData := []byte("Hello Go")
	if !bytes.Equal(asset.Data, expectedData) {
		t.Errorf("Expected data to be %q, got %q", expectedData, asset.Data)
	}
}

func TestReplacerTransformer_Transform_MultipleReplacements(t *testing.T) {
	asset := &Asset{
		Path: "/test.txt",
		Data: []byte("apple banana apple"),
	}
	transformer := ReplacerTransformer{
		Replacements: map[string]string{
			"apple":  "orange",
			"banana": "grape",
		},
	}

	err := transformer.Transform(asset)
	if err != nil {
		t.Fatalf("Transform returned an error: %v", err)
	}

	expectedData := []byte("orange grape orange")
	if !bytes.Equal(asset.Data, expectedData) {
		t.Errorf("Expected data to be %q, got %q", expectedData, asset.Data)
	}
}
