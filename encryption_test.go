package sitetools

import (
	"encoding/base64"
	"strings"
	"testing"
)

func newEncryptionTemplate() *Asset {
	return &Asset{
		Path: "/template.html",
		Data: []byte(`
			<html>
			<body>
				<form id="password-form">
					<input type="password" id="password" />
					<button type="submit">Unlock</button>
				</form>
				<div id="encrypted-content"></div>
			</body>
			</html>
		`),
	}
}

func TestEncryptionTransformer_UsesFixedSalt(t *testing.T) {
	template := newEncryptionTemplate()
	salt := []byte("0123456789abcdef0123456789abcdef")

	transformer := EncryptionTransformer{
		Template: template,
		Password: "test-password",
		Salt:     salt,
	}

	asset := &Asset{
		Path: "/test.html",
		Data: []byte("<html><body>Test content</body></html>"),
	}

	if err := transformer.Transform(asset); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedSalt := base64.StdEncoding.EncodeToString(salt)
	if !strings.Contains(string(asset.Data), expectedSalt) {
		t.Errorf("expected fixed salt to be embedded in output")
	}
}

func TestEncryptionTransformer_ReusesFixedSaltAcrossAssets(t *testing.T) {
	template := newEncryptionTemplate()
	salt := []byte("0123456789abcdef0123456789abcdef")

	transformer := EncryptionTransformer{
		Template: template,
		Password: "test-password",
		Salt:     salt,
	}

	asset1 := &Asset{
		Path: "/test1.html",
		Data: []byte("<html><body>Test content 1</body></html>"),
	}
	asset2 := &Asset{
		Path: "/test2.html",
		Data: []byte("<html><body>Test content 2</body></html>"),
	}

	if err := transformer.Transform(asset1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := transformer.Transform(asset2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedSalt := base64.StdEncoding.EncodeToString(salt)
	if !strings.Contains(string(asset1.Data), expectedSalt) {
		t.Errorf("expected fixed salt in first output")
	}
	if !strings.Contains(string(asset2.Data), expectedSalt) {
		t.Errorf("expected fixed salt in second output")
	}
}

func TestEncrypt_UsesRandomSaltWhenEmpty(t *testing.T) {
	data := []byte("test content")
	password := "test-password"
	iterations := 100000

	_, salt1, err := encrypt(data, password, iterations, nil)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}
	_, salt2, err := encrypt(data, password, iterations, nil)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if string(salt1) == string(salt2) {
		t.Error("expected random salts to differ when no salt is provided")
	}
}

func TestEncryptionTransformer_DefaultStorageModeIsNoOp(t *testing.T) {
	template := newEncryptionTemplate()

	transformer := EncryptionTransformer{
		Template: template,
		Password: "test-password",
	}

	asset := &Asset{
		Path: "/test.html",
		Data: []byte("<html><body>Test content</body></html>"),
	}

	if err := transformer.Transform(asset); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := string(asset.Data)

	if !strings.Contains(result, "const storage = noOpStorage") {
		t.Errorf("expected default storage mode to map storage to noOpStorage")
	}
	if !strings.Contains(result, "storage.getItem(derivedKeyStorageKey)") {
		t.Errorf("expected storage.getItem for derived key cache")
	}
	if !strings.Contains(result, "storage.setItem(derivedKeyStorageKey,") {
		t.Errorf("expected storage.setItem for derived key cache")
	}
}

func TestEncryptionTransformer_StorageModeSession(t *testing.T) {
	template := newEncryptionTemplate()

	transformer := EncryptionTransformer{
		Template:    template,
		Password:    "test-password",
		StorageMode: StoreSession,
	}

	asset := &Asset{
		Path: "/test.html",
		Data: []byte("<html><body>Test content</body></html>"),
	}

	if err := transformer.Transform(asset); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := string(asset.Data)

	if !strings.Contains(result, "const storage = window.sessionStorage") {
		t.Errorf("expected storage to map to window.sessionStorage")
	}
}

func TestEncryptionTransformer_StorageModeLocal(t *testing.T) {
	template := newEncryptionTemplate()

	transformer := EncryptionTransformer{
		Template:    template,
		Password:    "test-password",
		StorageMode: StoreLocal,
	}

	asset := &Asset{
		Path: "/test.html",
		Data: []byte("<html><body>Test content</body></html>"),
	}

	if err := transformer.Transform(asset); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := string(asset.Data)

	if !strings.Contains(result, "const storage = window.localStorage") {
		t.Errorf("expected storage to map to window.localStorage")
	}
}

func TestEncryptionTransformer_DerivedKeyCachingHooksPresent(t *testing.T) {
	template := newEncryptionTemplate()

	transformer := EncryptionTransformer{
		Template: template,
		Password: "test-password",
	}

	asset := &Asset{
		Path: "/test.html",
		Data: []byte("<html><body>Test content</body></html>"),
	}

	if err := transformer.Transform(asset); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := string(asset.Data)

	if !strings.Contains(result, "const derivedKeyStorageKey = 'derived_key_") {
		t.Errorf("expected derived key storage key declaration")
	}
	if !strings.Contains(result, "crypto.subtle.deriveBits(") {
		t.Errorf("expected deriveBits to be present in script")
	}
	if !strings.Contains(result, "storage.setItem(derivedKeyStorageKey,") {
		t.Errorf("expected derived key cache setItem call")
	}
}
