package sitetools

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"filippo.io/age"
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
		Algorithm: &AESGCMEncryption{
			Password: "test-password",
			Salt:     salt,
		},
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
		Algorithm: &AESGCMEncryption{
			Password: "test-password",
			Salt:     salt,
		},
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

func TestEncryptAESGCM_UsesRandomSaltWhenEmpty(t *testing.T) {
	data := []byte("test content")
	password := "test-password"
	iterations := 100000

	_, salt1, err := encryptAESGCM(data, password, iterations, nil)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}
	_, salt2, err := encryptAESGCM(data, password, iterations, nil)
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
		Template:  template,
		Algorithm: &AESGCMEncryption{Password: "test-password"},
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
		Algorithm:   &AESGCMEncryption{Password: "test-password"},
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
		Algorithm:   &AESGCMEncryption{Password: "test-password"},
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

func TestEncryptionTransformer_RequiresAlgorithm(t *testing.T) {
	template := newEncryptionTemplate()

	transformer := EncryptionTransformer{
		Template: template,
	}

	asset := &Asset{
		Path: "/test.html",
		Data: []byte("<html><body>x</body></html>"),
	}

	if err := transformer.Transform(asset); err == nil {
		t.Fatal("expected error when no Algorithm is configured")
	}
}

func TestAESGCMEncryption_RequiresPassword(t *testing.T) {
	template := newEncryptionTemplate()

	transformer := EncryptionTransformer{
		Template:  template,
		Algorithm: &AESGCMEncryption{},
	}

	asset := &Asset{
		Path: "/test.html",
		Data: []byte("<html><body>x</body></html>"),
	}

	if err := transformer.Transform(asset); err == nil {
		t.Fatal("expected error when AESGCMEncryption has no password")
	}
}

func TestAgeEncryption_RequiresPasswordOrRecipients(t *testing.T) {
	template := newEncryptionTemplate()

	transformer := EncryptionTransformer{
		Template:  template,
		Algorithm: &AgeEncryption{},
	}

	asset := &Asset{
		Path: "/test.html",
		Data: []byte("<html><body>secret</body></html>"),
	}

	if err := transformer.Transform(asset); err == nil {
		t.Fatal("expected error when neither password nor recipients are provided for age")
	}
}

func TestAgeEncryption_PassphraseProducesDecryptableCiphertext(t *testing.T) {
	template := newEncryptionTemplate()
	passphrase := "hunter2"
	plaintext := []byte("<html><body>top secret content</body></html>")

	transformer := EncryptionTransformer{
		Template:  template,
		Algorithm: &AgeEncryption{Password: passphrase},
	}

	asset := &Asset{
		Path: "/test.html",
		Data: append([]byte(nil), plaintext...),
	}

	if err := transformer.Transform(asset); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := string(asset.Data)

	if !strings.Contains(result, "window.ageEncryption") {
		t.Errorf("expected age decryption script to reference window.ageEncryption")
	}
	if !strings.Contains(result, "decrypter.addPassphrase(passphrase)") {
		t.Errorf("expected age decryption script to call addPassphrase")
	}
	// The vendored bundle exposes its global with this exact assignment.
	if !strings.Contains(result, "var ageEncryption=") {
		t.Errorf("expected vendored age library bundle to be inlined")
	}

	// Find the embedded base64 ciphertext and verify it actually decrypts.
	idx := strings.Index(result, "const encryptedBytes = toBytes('")
	if idx == -1 {
		t.Fatalf("could not find encryptedBytes assignment in output")
	}
	start := idx + len("const encryptedBytes = toBytes('")
	end := strings.Index(result[start:], "'")
	if end == -1 {
		t.Fatalf("could not find end of encryptedBytes literal")
	}
	b64 := result[start : start+end]
	ciphertext, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("could not base64-decode embedded ciphertext: %v", err)
	}

	identity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		t.Fatalf("could not build scrypt identity: %v", err)
	}
	r, err := age.Decrypt(bytes.NewReader(ciphertext), identity)
	if err != nil {
		t.Fatalf("age decrypt failed: %v", err)
	}
	var decrypted bytes.Buffer
	if _, err := decrypted.ReadFrom(r); err != nil {
		t.Fatalf("reading decrypted payload failed: %v", err)
	}
	if !bytes.Equal(decrypted.Bytes(), plaintext) {
		t.Errorf("decrypted bytes mismatch:\n got: %q\nwant: %q", decrypted.Bytes(), plaintext)
	}
}

func TestAgeEncryption_BundleIsInlinedAndNoExternalRefs(t *testing.T) {
	template := newEncryptionTemplate()

	transformer := EncryptionTransformer{
		Template:  template,
		Algorithm: &AgeEncryption{Password: "test"},
	}

	asset := &Asset{
		Path: "/test.html",
		Data: []byte("<html><body>x</body></html>"),
	}

	if err := transformer.Transform(asset); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := string(asset.Data)

	if strings.Contains(result, "esm.sh") || strings.Contains(result, "unpkg.com") || strings.Contains(result, "jsdelivr") {
		t.Errorf("expected no third-party CDN references in output")
	}
	if strings.Contains(result, `type="module"`) {
		t.Errorf("expected age script to no longer be a module (IIFE bundle is used)")
	}
}

func TestEncryptionTransformer_DerivedKeyCachingHooksPresent(t *testing.T) {
	template := newEncryptionTemplate()

	transformer := EncryptionTransformer{
		Template:  template,
		Algorithm: &AESGCMEncryption{Password: "test-password"},
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
