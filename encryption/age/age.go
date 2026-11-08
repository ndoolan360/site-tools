// Package age provides age-format page encryption and bundled browser decryption.
package age

import (
	"bytes"
	_ "embed"
	"fmt"

	"filippo.io/age"
	"go.doolan.dev/site-tools/encryption"
)

//go:embed assets/encryption_decrypt_script_age.js
var decryptScriptAgeTemplate []byte

//go:embed assets/age-encryption@0.3.0.decrypt-only.bundle.min.js
var ageEncryptionBundle []byte

// AgeEncryption encrypts content using the age file format (filippo.io/age).
// Decryption in the browser is performed with a vendored copy of the typage
// library (bundled into the page; no network access required at decrypt time).
type AgeEncryption struct {
	// Password, if non-empty, adds a scrypt passphrase recipient. The client-
	// side decryption script always prompts for and decrypts with a passphrase.
	Password string
	// Recipients lists additional age recipients (e.g. "age1..." X25519 keys).
	// The bundled decryption script does not handle X25519/SSH identities; if
	// you only configure Recipients you'll need a custom script template.
	Recipients []string
}

var _ encryption.EncryptionAlgorithm = (*AgeEncryption)(nil)
var _ encryption.PreambleProvider = (*AgeEncryption)(nil)

func (a *AgeEncryption) Encrypt(data []byte) ([]byte, map[string]string, error) {
	if a.Password == "" && len(a.Recipients) == 0 {
		return nil, nil, fmt.Errorf("age encryption requires a Password and/or Recipients")
	}

	var recipients []age.Recipient
	if a.Password != "" {
		r, err := age.NewScryptRecipient(a.Password)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid scrypt passphrase: %w", err)
		}
		recipients = append(recipients, r)
	}
	for _, s := range a.Recipients {
		r, err := age.ParseX25519Recipient(s)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid age recipient %q: %w", s, err)
		}
		recipients = append(recipients, r)
	}

	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, recipients...)
	if err != nil {
		return nil, nil, err
	}
	if _, err := w.Write(data); err != nil {
		return nil, nil, err
	}
	if err := w.Close(); err != nil {
		return nil, nil, err
	}

	return buf.Bytes(), nil, nil
}

func (a *AgeEncryption) ScriptTemplate() []byte { return decryptScriptAgeTemplate }

// PreambleScript returns the vendored, minified typage IIFE bundle which
// exposes the library on window.ageEncryption.
func (a *AgeEncryption) PreambleScript() []byte { return ageEncryptionBundle }
