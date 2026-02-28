package sitetools

import (
	"path"
	"reflect"
	"strings"
	"testing"
)

func TestPop(t *testing.T) {
	asset1 := newTestAsset("/path/to/file1.txt", "content1", map[string]any{"tag": "odd"})
	asset2 := newTestAsset("/path/to/file2.txt", "content2", map[string]any{"tag": "even"})
	asset3 := newTestAsset("/path/to/file3.txt", "content3", map[string]any{"tag": "odd"})

	assets := Assets{asset1, asset2, asset3}
	originalAssets := Assets{asset1, asset2, asset3}

	// Filter to pop assets with tag "odd"
	oddTagFilter := func(a Asset) bool {
		tag, ok := a.Meta["tag"].(string)
		return ok && tag == "odd"
	}

	popped := assets.Pop(oddTagFilter)

	// Check popped assets
	expectedPopped := Assets{asset1, asset3}
	if !reflect.DeepEqual(popped, expectedPopped) {
		t.Errorf("Pop() popped = %v, want %v", popped, expectedPopped)
	}

	// Check remaining assets in the original slice
	expectedRemaining := Assets{asset2}
	if !reflect.DeepEqual(assets, expectedRemaining) {
		t.Errorf("Pop() remaining assets = %v, want %v", assets, expectedRemaining)
	}

	// Test with no filters (should pop nothing)
	assets = Assets{asset1, asset2, asset3}
	popped = assets.Pop()
	if len(popped) != 0 {
		t.Errorf("Pop() with no filters popped = %v, want empty", popped)
	}
	if !reflect.DeepEqual(assets, originalAssets) {
		t.Errorf("Pop() with no filters modified original assets = %v, want %v", assets, originalAssets)
	}

	// Test with a filter that matches nothing
	assets = Assets{asset1, asset2, asset3}
	neverPopFilter := func(a Asset) bool { return false }
	popped = assets.Pop(neverPopFilter)
	if len(popped) != 0 {
		t.Errorf("Pop() with neverPopFilter popped = %v, want empty", popped)
	}
	if !reflect.DeepEqual(assets, originalAssets) {
		t.Errorf("Pop() with neverPopFilter modified original assets = %v, want %v", assets, originalAssets)
	}

	// Test with a filter that matches everything
	assets = Assets{asset1, asset2, asset3}
	alwaysPopFilter := func(a Asset) bool { return true }
	popped = assets.Pop(alwaysPopFilter)
	if !reflect.DeepEqual(popped, originalAssets) {
		t.Errorf("Pop() with alwaysPopFilter popped = %v, want %v", popped, originalAssets)
	}
	if len(assets) != 0 {
		t.Errorf("Pop() with alwaysPopFilter remaining assets = %v, want empty", assets)
	}
}

func TestFilter(t *testing.T) {
	asset1 := newTestAsset("/path/to/file1.txt", "content1", map[string]any{"tag": "odd", "type": "A"})
	asset2 := newTestAsset("/path/to/file2.txt", "content2", map[string]any{"tag": "even", "type": "B"})
	asset3 := newTestAsset("/path/to/file3.txt", "content3", map[string]any{"tag": "odd", "type": "A"})

	assets := Assets{asset1, asset2, asset3}
	originalAssets := Assets{asset1, asset2, asset3}

	// Filter to get assets with tag "odd"
	oddTagFilter := func(a Asset) bool {
		tag, ok := a.Meta["tag"].(string)
		return ok && tag == "odd"
	}

	filtered := assets.Filter(oddTagFilter)

	// Check filtered assets
	expectedFiltered := Assets{asset1, asset3}
	if !reflect.DeepEqual(filtered, expectedFiltered) {
		t.Errorf("Filter() filtered = %v, want %v", filtered, expectedFiltered)
	}

	// Check that the original slice is unchanged
	if !reflect.DeepEqual(assets, originalAssets) {
		t.Errorf("Filter() modified original assets = %v, want %v", assets, originalAssets)
	}

	// Test with multiple filters
	typeAFilter := func(a Asset) bool {
		typeVal, ok := a.Meta["type"].(string)
		return ok && typeVal == "A"
	}
	filteredMultiple := assets.Filter(oddTagFilter, typeAFilter)
	expectedFilteredMultiple := Assets{asset1, asset3}
	if !reflect.DeepEqual(filteredMultiple, expectedFilteredMultiple) {
		t.Errorf("Filter() with multiple filters = %v, want %v", filteredMultiple, expectedFilteredMultiple)
	}

	// Test with no filters (should return all assets)
	filteredNone := assets.Filter()
	if !reflect.DeepEqual(filteredNone, originalAssets) {
		t.Errorf("Filter() with no filters = %v, want %v", filteredNone, originalAssets)
	}

	// Test with a filter that matches nothing
	neverFilter := func(a Asset) bool { return false }
	filteredNever := assets.Filter(neverFilter)
	if len(filteredNever) != 0 {
		t.Errorf("Filter() with neverFilter = %v, want empty", filteredNever)
	}
}

func TestWithParentDir(t *testing.T) {
	assets := Assets{
		&Asset{Path: "/tools/go/file1.txt"},
		&Asset{Path: "/tools/go/subdir/file2.txt"},
		&Asset{Path: "/other/file3.txt"},
	}

	filtered := assets.Filter(WithParentDir("/tools/go"))
	if len(filtered) != 2 {
		t.Errorf("Expected 2 assets, got %d", len(filtered))
	}

	for _, asset := range filtered {
		if !strings.HasPrefix(asset.Path, "/tools/go") {
			t.Errorf("Asset %s should be in /tools/go", asset.Path)
		}
	}
}

func TestWithPath(t *testing.T) {
	assets := Assets{
		&Asset{Path: "/test/file1.txt"},
		&Asset{Path: "/test/file2.txt"},
		&Asset{Path: "/test/file3.md"},
	}

	filtered := assets.Filter(WithPath("/test/file2.txt"))
	if len(filtered) != 1 {
		t.Errorf("Expected 1 asset, got %d", len(filtered))
	}

	if filtered[0].Path != "/test/file2.txt" {
		t.Errorf("Expected asset path to be /test/file2.txt, got %s", filtered[0].Path)
	}
}

func TestWithoutPath(t *testing.T) {
	assets := Assets{
		&Asset{Path: "/test/file1.txt"},
		&Asset{Path: "/test/file2.txt"},
		&Asset{Path: "/test/file3.md"},
	}

	filtered := assets.Filter(WithoutPath("/test/file2.txt"))
	if len(filtered) != 2 {
		t.Errorf("Expected 2 assets, got %d", len(filtered))
	}

	for _, asset := range filtered {
		if asset.Path == "/test/file2.txt" {
			t.Error("Filtered asset should not be /test/file2.txt")
		}
	}
}

func TestWithExtensions(t *testing.T) {
	assets := Assets{
		&Asset{Path: "/test/file1.txt"},
		&Asset{Path: "/test/file2.txt"},
		&Asset{Path: "/test/file3.md"},
	}

	filtered := assets.Filter(WithExtensions(".txt"))
	if len(filtered) != 2 {
		t.Errorf("Expected 2 assets, got %d", len(filtered))
	}

	for _, asset := range filtered {
		if path.Ext(asset.Path) != ".txt" {
			t.Errorf("Asset %s should have .txt extension", asset.Path)
		}
	}
}

func TestWithoutExtensions(t *testing.T) {
	assets := Assets{
		&Asset{Path: "/test/file1.txt"},
		&Asset{Path: "/test/file2.txt"},
		&Asset{Path: "/test/file3.md"},
	}

	filtered := assets.Filter(WithoutExtensions(".txt"))
	if len(filtered) != 1 {
		t.Errorf("Expected 1 asset, got %d", len(filtered))
	}

	if path.Ext(filtered[0].Path) != ".md" {
		t.Errorf("Expected asset to have .md extension, got %s", filtered[0].Path)
	}
}

func TestWithMeta(t *testing.T) {
	assets := Assets{
		&Asset{Path: "/test/file1.txt", Meta: map[string]any{"IsDraft": false}},
		&Asset{Path: "/test/file2.txt", Meta: map[string]any{"IsDraft": true}},
		&Asset{Path: "/test/file3.txt", Meta: map[string]any{"IsDraft": "false"}},
		&Asset{Path: "/test/file4.txt", Meta: map[string]any{"IsDraft": "true"}},
		&Asset{Path: "/test/file5.txt", Meta: map[string]any{"IsDraft": "  FALSE  "}},
		&Asset{Path: "/test/file6.txt", Meta: map[string]any{"IsDraft": "  TRUE  "}},
		&Asset{Path: "/test/file7.md"},
	}

	filtered := assets.Filter(WithMeta("IsDraft"))
	if len(filtered) != 3 {
		t.Errorf("Expected 3 assets, got %d", len(filtered))
	}

	if filtered[0].Path != "/test/file2.txt" {
		t.Errorf("Expected /test/file2.txt, got %s", filtered[0].Path)
	}
	if filtered[1].Path != "/test/file4.txt" {
		t.Errorf("Expected /test/file4.txt, got %s", filtered[1].Path)
	}
	if filtered[2].Path != "/test/file6.txt" {
		t.Errorf("Expected /test/file6.txt, got %s", filtered[2].Path)
	}
}

func TestWithoutMeta(t *testing.T) {
	assets := Assets{
		&Asset{Path: "/test/file1.txt", Meta: map[string]any{"IsDraft": false}},
		&Asset{Path: "/test/file2.txt", Meta: map[string]any{"IsDraft": true}},
		&Asset{Path: "/test/file3.txt", Meta: map[string]any{"IsDraft": "false"}},
		&Asset{Path: "/test/file4.txt", Meta: map[string]any{"IsDraft": "true"}},
		&Asset{Path: "/test/file5.txt", Meta: map[string]any{"IsDraft": "  FALSE  "}},
		&Asset{Path: "/test/file6.txt", Meta: map[string]any{"IsDraft": "  TRUE  "}},
		&Asset{Path: "/test/file7.md"},
	}

	filtered := assets.Filter(WithoutMeta("IsDraft"))
	if len(filtered) != 4 {
		t.Errorf("Expected 4 asset, got %d", len(filtered))
	}

	for _, asset := range filtered {
		if asset.Path == "/test/file2.txt" || asset.Path == "/test/file4.txt" || asset.Path == "/test/file6.txt" {
			t.Error("Filtered asset should not be a draft")
		}
	}
}

func TestWithMimeType(t *testing.T) {
	assets := Assets{
		&Asset{Path: "/test/file1.txt"},
		&Asset{Path: "/test/file2.md"},
		&Asset{Path: "/test/file3.html"},
	}

	filtered := assets.Filter(WithMimeType("text/plain"))
	if len(filtered) != 1 {
		t.Errorf("Expected 1 asset, got %d", len(filtered))
	}

	if filtered[0].Path != "/test/file1.txt" {
		t.Errorf("Expected /test/file1.txt, got %s", filtered[0].Path)
	}
}

func TestWithMimeType_TopLevel(t *testing.T) {
	assets := Assets{
		&Asset{Path: "/test/file1.txt"},
		&Asset{Path: "/test/file2.md"},
		&Asset{Path: "/test/file3.png"},
	}

	filtered := assets.Filter(WithMimeType("text/*"))
	if len(filtered) != 2 {
		t.Errorf("Expected 2 assets, got %d", len(filtered))
	}
}

func TestWithoutMimeType(t *testing.T) {
	assets := Assets{
		&Asset{Path: "/test/file1.txt"},
		&Asset{Path: "/test/file2.md"},
		&Asset{Path: "/test/file3.html"},
	}

	filtered := assets.Filter(WithoutMimeType("text/plain"))
	if len(filtered) != 2 {
		t.Errorf("Expected 2 assets, got %d", len(filtered))
	}

	for _, asset := range filtered {
		if asset.Path == "/test/file1.txt" {
			t.Error("Filtered asset should not be /test/file1.txt")
		}
	}
}

func TestWithoutMimeType_TopLevel(t *testing.T) {
	assets := Assets{
		&Asset{Path: "/test/file1.txt"},
		&Asset{Path: "/test/file2.md"},
		&Asset{Path: "/test/file3.png"},
	}

	filtered := assets.Filter(WithoutMimeType("text/*"))
	if len(filtered) != 1 {
		t.Errorf("Expected 1 asset, got %d", len(filtered))
	}

	if filtered[0].Path != "/test/file3.png" {
		t.Errorf("Expected /test/file3.png, got %s", filtered[0].Path)
	}
}
