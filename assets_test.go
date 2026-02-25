package sitetools

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func newTestAsset(path string, data string, meta map[string]any) *Asset {
	return &Asset{Path: path, Data: []byte(data), Meta: meta}
}

type MockTransformer struct {
	TransformFunc func(*Asset) error
	CalledCount   int
}

func (m *MockTransformer) Transform(asset *Asset) error {
	m.CalledCount++
	if m.TransformFunc != nil {
		return m.TransformFunc(asset)
	}
	return nil
}

func TestAssets_Add(t *testing.T) {
	assets := Assets{}

	asset1 := newTestAsset("/test1.txt", "data", nil)
	asset2 := newTestAsset("test2.txt", "data", nil)
	assets.Add(*asset1, *asset2)

	if len(assets) != 2 {
		t.Fatalf("expected 2 asset, got %d", len(assets))
	}
	if assets[0].Path != "/test1.txt" {
		t.Errorf("Asset Path = %s, want /test.txt", assets[0].Path)
	}
	if string(assets[0].Data) != "data" {
		t.Errorf("Asset Data = %s, want data", string(assets[0].Data))
	}
	if assets[0].Meta != nil {
		t.Errorf("Asset Meta = %v, want nil", assets[0].Meta)
	}
	if assets[1].Path != "/test2.txt" {
		t.Errorf("Asset Path = %s, want test2.txt", assets[1].Path)
	}
	if string(assets[1].Data) != "data" {
		t.Errorf("Asset Data = %s, want data", string(assets[1].Data))
	}
	if assets[1].Meta != nil {
		t.Errorf("Asset Meta = %v, want nil", assets[1].Meta)
	}
}

func TestAssets_Transform(t *testing.T) {
	asset1 := newTestAsset("1.txt", "content", nil)
	asset2 := newTestAsset("2.txt", "content", nil)
	assets := Assets{asset1, asset2}

	transformer1 := &MockTransformer{
		TransformFunc: func(a *Asset) error {
			if a.Meta == nil {
				a.Meta = make(map[string]any)
			}
			a.Meta["transformedBy"] = "transformer1"
			return nil
		},
	}

	transformer2 := &MockTransformer{
		TransformFunc: func(a *Asset) error {
			if a.Meta == nil {
				a.Meta = make(map[string]any)
			}
			if val, ok := a.Meta["transformedBy"].(string); ok {
				a.Meta["transformedBy"] = val + ", transformer2"
			} else {
				a.Meta["transformedBy"] = "transformer2"
			}
			return nil
		},
	}

	err := assets.Transform(transformer1, transformer2)
	if err != nil {
		t.Fatalf("Transform() returned error: %v", err)
	}

	if transformer1.CalledCount != 2 {
		t.Errorf("transformer1 was called %d times, want 2", transformer1.CalledCount)
	}
	if transformer2.CalledCount != 2 {
		t.Errorf("transformer2 was called %d times, want 2", transformer2.CalledCount)
	}

	for i, asset := range assets {
		expectedMeta := "transformer1, transformer2"
		if meta, ok := asset.Meta["transformedBy"].(string); !ok || meta != expectedMeta {
			t.Errorf("Asset %d Meta[\"transformedBy\"] = %v, want %v", i, asset.Meta["transformedBy"], expectedMeta)
		}
	}

	failingTransformer := &MockTransformer{
		TransformFunc: func(a *Asset) error {
			return os.ErrPermission
		},
	}
	assetsSingle := Assets{newTestAsset("fail.txt", "", nil)}
	err = assetsSingle.Transform(failingTransformer)
	if err == nil {
		t.Errorf("Transform() with failing transformer did not return an error")
	} else if err != os.ErrPermission {
		t.Errorf("Transform() with failing transformer returned wrong error: got %v, want %v", err, os.ErrPermission)
	}
}

func TestAssets_Write(t *testing.T) {
	tmpDir := t.TempDir()

	asset1 := newTestAsset("/file1.txt", "content1", nil)
	asset2 := newTestAsset("/subdir/file2.txt", "content2", nil)

	assets := Assets{asset1, asset2}

	err := assets.Write(tmpDir)
	if err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}

	diskPath1 := tmpDir + asset1.Path
	content1, err := os.ReadFile(diskPath1)
	if err != nil {
		t.Errorf("Failed to read %s: %v", diskPath1, err)
	}
	if string(content1) != string(asset1.Data) {
		t.Errorf("Content of %s = %s, want %s", diskPath1, string(content1), string(asset1.Data))
	}

	diskPath2 := tmpDir + asset2.Path
	content2, err := os.ReadFile(diskPath2)
	if err != nil {
		t.Errorf("Failed to read %s: %v", diskPath2, err)
	}
	if string(content2) != string(asset2.Data) {
		t.Errorf("Content of %s = %s, want %s", diskPath2, string(content2), string(asset2.Data))
	}

	if len(assets) > 0 {
		filePathAsDir := tmpDir + "/blocked_dir"
		err = os.WriteFile(filePathAsDir, []byte("i am a file"), 0644)
		if err != nil {
			t.Fatalf("Could not create blocking file: %v", err)
		}

		asset3 := newTestAsset("/blocked_dir/file3.txt", "content3", nil)
		assetsBlocking := Assets{asset3}
		err = assetsBlocking.Write(tmpDir)
		if err == nil {
			t.Errorf("Write() did not return error when path component (blocked_dir) is a file")
		}
	}
}

func TestAssets_ToMap(t *testing.T) {
	asset1 := newTestAsset("/path/to/file1.txt", "content1", map[string]any{"id": "id1", "tag": "odd"})
	asset2 := newTestAsset("/path/to/file2.txt", "content2", map[string]any{"id": "id2", "tag": "even"})
	asset3 := newTestAsset("/path/to/file3.txt", "content3", map[string]any{"tag": "odd"})
	asset4 := newTestAsset("/path/to/file4.txt", "content4", map[string]any{"id": 123})

	assets := Assets{asset1, asset2, asset3, asset4}

	assetMap := assets.ToMap("id")

	expectedMap := map[string]*Asset{
		"id1": asset1,
		"id2": asset2,
	}

	if !reflect.DeepEqual(assetMap, expectedMap) {
		t.Errorf("ToMap() map = %v, want %v", assetMap, expectedMap)
	}

	// Test with a key that doesn't exist
	emptyMap := assets.ToMap("nonexistent_key")
	if len(emptyMap) != 0 {
		t.Errorf("ToMap() with nonexistent key = %v, want empty map", emptyMap)
	}
}

func TestAssets_SetMetaFunc(t *testing.T) {
	asset1 := newTestAsset("/path/file1.txt", "data1", map[string]any{"id": "1"})
	asset2 := newTestAsset("/another/file2.txt", "data2", nil)
	asset3 := newTestAsset("/short.txt", "data3", map[string]any{})

	assets := Assets{asset1, asset2, asset3}

	pathPrefixFunc := func(a Asset) string {
		return filepath.Dir(a.Path)
	}

	assets.SetMetaFunc("dir", pathPrefixFunc)

	// Check asset1
	if dir, ok := asset1.Meta["dir"].(string); !ok || dir != "/path" {
		t.Errorf("SetMetaFunc() asset1.Meta[\"dir\"] = %v, want %v", asset1.Meta["dir"], "/path")
	}

	// Check asset2 (meta was nil, should be skipped)
	if asset2.Meta != nil {
		if _, ok := asset2.Meta["dir"]; ok {
			t.Errorf("SetMetaFunc() asset2.Meta[\"dir\"] was added, but Meta was initially nil and should be skipped. Meta: %v", asset2.Meta)
		}
	}

	// Check asset3
	if dir, ok := asset3.Meta["dir"].(string); !ok || dir != "/" { // filepath.Dir("/short.txt") is "/"
		t.Errorf("SetMetaFunc() asset3.Meta[\"dir\"] = %v, want %v", asset3.Meta["dir"], "/")
	}
}

func TestAssets_AddToMeta(t *testing.T) {
	asset1 := newTestAsset("1.txt", "", map[string]any{"key1": "val1"})
	asset2 := newTestAsset("2.txt", "", nil)              // Meta is nil
	asset3 := newTestAsset("3.txt", "", map[string]any{}) // Meta is empty

	assets := Assets{asset1, asset2, asset3}
	assets.AddToMeta("newKey", "newValue")

	// Check asset1
	if val, ok := asset1.Meta["newKey"].(string); !ok || val != "newValue" {
		t.Errorf("AddToMeta() asset1.Meta[\"newKey\"] = %v, want %v", asset1.Meta["newKey"], "newValue")
	}
	if val, ok := asset1.Meta["key1"].(string); !ok || val != "val1" {
		t.Errorf("AddToMeta() asset1.Meta[\"key1\"] was changed or removed")
	}

	// Check asset2 (meta was nil, should be skipped)
	if asset2.Meta != nil {
		if _, ok := asset2.Meta["newKey"]; ok {
			t.Errorf("AddToMeta() asset2.Meta[\"newKey\"] was added, but Meta was initially nil and should be skipped. Meta: %v", asset2.Meta)
		}
	}

	// Check asset3
	if val, ok := asset3.Meta["newKey"].(string); !ok || val != "newValue" {
		t.Errorf("AddToMeta() asset3.Meta[\"newKey\"] = %v, want %v", asset3.Meta["newKey"], "newValue")
	}
}
