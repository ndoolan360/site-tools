package age

import (
	"bytes"
	"encoding/base64"
	"io"
	"strings"
	"testing"

	"filippo.io/age"
	sitetools "go.doolan.dev/site-tools"
	"go.doolan.dev/site-tools/encryption"
)

func newEncryptionTemplate() *sitetools.Asset {
	return &sitetools.Asset{
		Path: "/template.html",
		Data: []byte(`<html><body>
<form id="password-form"><input type="password" id="password" /><button type="submit">Unlock</button></form>
<div id="encrypted-content"></div>
</body></html>`),
	}
}

func TestAgeEncryption_RequiresPasswordOrRecipients(t *testing.T) {
	transformer := encryption.EncryptionTransformer{
		Template:  newEncryptionTemplate(),
		Algorithm: &AgeEncryption{},
	}
	asset := &sitetools.Asset{Path: "/test.html", Data: []byte("secret")}
	if err := transformer.Transform(asset); err == nil {
		t.Fatal("expected error when neither password nor recipients are provided for age")
	}
}

func TestAgeEncryption_PassphraseProducesDecryptableCiphertext(t *testing.T) {
	passphrase := "hunter2"
	plaintext := []byte("<html><body>top secret content</body></html>")
	transformer := encryption.EncryptionTransformer{
		Template:  newEncryptionTemplate(),
		Algorithm: &AgeEncryption{Password: passphrase},
	}
	asset := &sitetools.Asset{Path: "/test.html", Data: bytes.Clone(plaintext)}
	if err := transformer.Transform(asset); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result := string(asset.Data)
	for _, want := range []string{"window.ageEncryption", "decrypter.addPassphrase(passphrase)", "var ageEncryption="} {
		if !strings.Contains(result, want) {
			t.Errorf("expected age output to contain %q", want)
		}
	}

	// Verify the ciphertext embedded in the page can be decrypted with age.
	_, literal, ok := strings.Cut(result, "const encryptedBytes = toBytes('")
	if !ok {
		t.Fatal("could not find encryptedBytes assignment in output")
	}
	b64, _, ok := strings.Cut(literal, "'")
	if !ok {
		t.Fatal("could not find end of encryptedBytes literal")
	}
	ciphertext, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("could not base64-decode embedded ciphertext: %v", err)
	}
	identity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		t.Fatalf("could not build scrypt identity: %v", err)
	}
	assertDecrypts(t, ciphertext, plaintext, identity)
}

func TestAgeEncryption_BundleIsInlinedAndNoExternalRefs(t *testing.T) {
	transformer := encryption.EncryptionTransformer{
		Template:  newEncryptionTemplate(),
		Algorithm: &AgeEncryption{Password: "test"},
	}
	asset := &sitetools.Asset{Path: "/test.html", Data: []byte("secret")}
	if err := transformer.Transform(asset); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result := string(asset.Data)
	for _, external := range []string{"esm.sh", "unpkg.com", "jsdelivr", `type="module"`} {
		if strings.Contains(result, external) {
			t.Errorf("expected no external or module-script references; found %q", external)
		}
	}
	if !strings.Contains(result, "var ageEncryption=") {
		t.Error("expected vendored age library bundle to be inlined")
	}
}

func TestAgeEncryption_RecipientProducesDecryptableCiphertext(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	algorithm := &AgeEncryption{Recipients: []string{identity.Recipient().String()}}
	plaintext := []byte("recipient-only payload")
	ciphertext, _, err := algorithm.Encrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	assertDecrypts(t, ciphertext, plaintext, identity)
}

func TestAgeEncryption_InvalidRecipient(t *testing.T) {
	_, _, err := (&AgeEncryption{Recipients: []string{"invalid"}}).Encrypt([]byte("secret"))
	if err == nil || !strings.Contains(err.Error(), "invalid age recipient") {
		t.Fatalf("expected invalid recipient error, got %v", err)
	}
}

func assertDecrypts(t *testing.T, ciphertext, plaintext []byte, identity age.Identity) {
	t.Helper()
	r, err := age.Decrypt(bytes.NewReader(ciphertext), identity)
	if err != nil {
		t.Fatalf("age decrypt failed: %v", err)
	}
	decrypted, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading decrypted payload failed: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted bytes mismatch:\n got: %q\nwant: %q", decrypted, plaintext)
	}
}
