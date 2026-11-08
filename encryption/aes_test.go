package encryption

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"

	"testing"
)

func TestEncryptAESGCM_PBKDF2Compatibility(t *testing.T) {
	plaintext := []byte("compatible browser ciphertext")
	salt := []byte("salt")
	ciphertext, returnedSalt, err := encryptAESGCM(plaintext, "password", 1, salt)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(returnedSalt, salt) {
		t.Fatalf("salt changed: got %q, want %q", returnedSalt, salt)
	}

	// PBKDF2-HMAC-SHA256("password", "salt", 1, 32) is a known test vector.
	// Use its fixed key to check compatibility independently of the KDF implementation.
	key, err := hex.DecodeString("120fb6cffcf8b32c43e7225256c4f837a86548c92ccc35480805987cb70be17b")
	if err != nil {
		t.Fatal(err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	if len(ciphertext) < gcm.NonceSize()+gcm.Overhead() {
		t.Fatalf("ciphertext too short: %d bytes", len(ciphertext))
	}
	decrypted, err := gcm.Open(nil, ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():], nil)
	if err != nil {
		t.Fatalf("ciphertext is incompatible with PBKDF2-HMAC-SHA256 test vector: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
	returnedSalt[0] = 'S'
	if string(salt) != "salt" {
		t.Error("encryption returned an alias to the caller's salt")
	}
}

func TestAESGCMEncryption_DefaultIterations(t *testing.T) {
	_, replacements, err := (&AESGCMEncryption{Password: "password"}).Encrypt([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if got := replacements["{{.ITERATIONS}}"]; got != "600000" {
		t.Errorf("default iterations = %q, want 600000", got)
	}
}
