package sitetools

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"

	"filippo.io/age"
	"golang.org/x/crypto/pbkdf2"
)

//go:embed assets/encryption_decrypt_script.js
var decryptScriptTemplate []byte

//go:embed assets/encryption_decrypt_script_age.js
var decryptScriptAgeTemplate []byte

//go:embed assets/age-encryption@0.3.0.decrypt-only.bundle.min.js
var ageEncryptionBundle []byte

type StorageMode string

const (
	StoreNone    StorageMode = "noOpStorage"
	StoreLocal   StorageMode = "window.localStorage"
	StoreSession StorageMode = "window.sessionStorage"
)

// EncryptionAlgorithm is the pluggable strategy used by EncryptionTransformer.
// Implementations encapsulate both the server-side encryption and the
// client-side decryption script.
type EncryptionAlgorithm interface {
	// Encrypt produces the raw ciphertext for data along with algorithm-specific
	// replacements to be applied to the decryption script template.
	// EncryptionTransformer is responsible for adding the common replacements
	// (ENCRYPTED_DATA, ENCRYPTED_DATA_PREFIX, IDs, STORAGE_MODE).
	Encrypt(data []byte) (ciphertext []byte, replacements map[string]string, err error)
	// ScriptTemplate returns the client-side decryption JS template.
	ScriptTemplate() []byte
}

// PreambleProvider is an optional interface that EncryptionAlgorithm
// implementations may also implement to inject an extra <script> tag before
// the decryption script (e.g. a bundled library that the decryption script
// depends on). Returned bytes are not subject to template replacements.
type PreambleProvider interface {
	PreambleScript() []byte
}

// EncryptionTransformer encrypts page content with client-side decryption.
//
// All algorithm-specific configuration lives on the Algorithm value (see
// AESGCMEncryption and AgeEncryption). The fields on the transformer itself
// only configure the surrounding template wiring.
type EncryptionTransformer struct {
	Template *Asset
	// Algorithm selects the encryption scheme. Required.
	Algorithm EncryptionAlgorithm
	// PasswordInputID is the ID of the password input element in the template (default: "password")
	PasswordInputID string
	// FormID is the ID of the form element in the template (default: "password-form")
	FormID string
	// ContentID is the ID of the element where decrypted content will be placed (default: "encrypted-content")
	ContentID string
	// StorageMode determines where the decrypted contents are stored (default: StoreNone)
	StorageMode StorageMode
	// MinifyScript determines if the script injected into the password template is minified (default: false)
	MinifyScript bool
}

func (t EncryptionTransformer) Transform(asset *Asset) error {
	if t.Template == nil {
		return fmt.Errorf("encryption template is required")
	}
	if t.Algorithm == nil {
		return fmt.Errorf("encryption algorithm is required")
	}

	passwordInputID := t.PasswordInputID
	formID := t.FormID
	contentID := t.ContentID
	storageMode := t.StorageMode

	if passwordInputID == "" {
		passwordInputID = "password"
	}
	if formID == "" {
		formID = "password-form"
	}
	if contentID == "" {
		contentID = "encrypted-content"
	}
	if storageMode == "" {
		storageMode = StoreNone
	}

	// Validate template has required elements
	templateStr := string(t.Template.Data)
	for _, requiredID := range []string{passwordInputID, formID, contentID} {
		if !strings.Contains(templateStr, fmt.Sprintf(`id="%s"`, requiredID)) {
			return fmt.Errorf("encryption template must contain element with id '%s'", requiredID)
		}
	}

	ciphertext, algoReplacements, err := t.Algorithm.Encrypt(asset.Data)
	if err != nil {
		return fmt.Errorf("failed to encrypt asset: %w", err)
	}

	encryptedBase64 := base64.StdEncoding.EncodeToString(ciphertext)

	replacements := map[string]string{
		"{{.ENCRYPTED_DATA}}":        encryptedBase64,
		"{{.ENCRYPTED_DATA_PREFIX}}": encryptedBase64[:min(32, len(encryptedBase64))],
		"{{.PASSWORD_INPUT_ID}}":     passwordInputID,
		"{{.FORM_ID}}":               formID,
		"{{.CONTENT_ID}}":            contentID,
		"{{.STORAGE_MODE}}":          string(storageMode),
	}
	for k, v := range algoReplacements {
		replacements[k] = v
	}

	decryptionScript, err := renderDecryptionScript(t.Algorithm.ScriptTemplate(), replacements, t.MinifyScript)
	if err != nil {
		return fmt.Errorf("failed to generate decryption script: %w", err)
	}

	var scriptTag strings.Builder
	if p, ok := t.Algorithm.(PreambleProvider); ok {
		if preamble := p.PreambleScript(); len(preamble) > 0 {
			scriptTag.WriteString("\n<script>\n")
			scriptTag.WriteString(escapeScriptBody(string(preamble)))
			scriptTag.WriteString("\n</script>")
		}
	}
	scriptTag.WriteString("\n<script>")
	scriptTag.WriteString(decryptionScript)
	scriptTag.WriteString("\n</script>\n")

	asset.Data = []byte(injectBeforeClose(templateStr, scriptTag.String()))
	return nil
}

// injectBeforeClose inserts content just before the last </body> tag in html,
// falling back to </html>, falling back to appending at the end.
func injectBeforeClose(html, content string) string {
	for _, tag := range []string{"</body>", "</html>"} {
		if idx := strings.LastIndex(html, tag); idx != -1 {
			return html[:idx] + content + html[idx:]
		}
	}
	return html + content
}

// escapeScriptBody escapes any literal `</script` sequences in a script body
// so it can be safely inlined inside a <script> tag.
func escapeScriptBody(s string) string {
	// Case-insensitive replace of </script with <\/script to avoid prematurely
	// terminating the surrounding tag.
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if i+8 <= len(s) && strings.EqualFold(s[i:i+8], "</script") {
			b.WriteString("<\\/script")
			i += 8
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// renderDecryptionScript applies replacements to the given JS template and
// optionally minifies the result.
func renderDecryptionScript(template []byte, replacements map[string]string, minify bool) (string, error) {
	scriptAsset := &Asset{
		Path: "encryption_decrypt_script.js",
		Data: template,
	}
	if err := (ReplacerTransformer{Replacements: replacements}).Transform(scriptAsset); err != nil {
		return "", err
	}
	if minify {
		if err := (MinifyTransformer{}).Transform(scriptAsset); err != nil {
			return "", err
		}
	}
	return string(scriptAsset.Data), nil
}

// RandomSalt returns a 32-byte random salt suitable for PBKDF2.
// Use this if you want to supply a fixed salt for AESGCMEncryption.
func RandomSalt() ([]byte, error) {
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// AESGCMEncryption encrypts content with AES-GCM using a PBKDF2-derived key.
// Decryption in the browser only requires the Web Crypto API.
type AESGCMEncryption struct {
	// Password is required. It is NOT stored in the output - only used during encryption.
	Password string
	// Iterations is the number of PBKDF2 iterations for key derivation (default: 600000).
	// Higher values increase security but also increase decryption time in the browser.
	Iterations int
	// Salt is optional. If provided, it is reused for all pages encrypted by this algorithm.
	// If empty, a random salt is generated per page (default behavior).
	// Setting this to a fixed value is less secure but allows for faster decryption, consistent
	// encrypted output across builds, and sharing the same password across multiple pages if they
	// also share the same iterations value.
	Salt []byte
}

func (a *AESGCMEncryption) Encrypt(data []byte) ([]byte, map[string]string, error) {
	if a.Password == "" {
		return nil, nil, fmt.Errorf("password is required for AES-GCM encryption")
	}
	iterations := a.Iterations
	if iterations == 0 {
		iterations = 600000
	}
	ciphertext, salt, err := encryptAESGCM(data, a.Password, iterations, a.Salt)
	if err != nil {
		return nil, nil, err
	}
	return ciphertext, map[string]string{
		"{{.SALT}}":       base64.StdEncoding.EncodeToString(salt),
		"{{.ITERATIONS}}": strconv.Itoa(iterations),
	}, nil
}

func (a *AESGCMEncryption) ScriptTemplate() []byte { return decryptScriptTemplate }

// encryptAESGCM encrypts data using AES-GCM with PBKDF2 key derivation.
func encryptAESGCM(data []byte, password string, iterations int, salt []byte) ([]byte, []byte, error) {
	if len(salt) == 0 {
		var err error
		salt, err = RandomSalt()
		if err != nil {
			return nil, nil, err
		}
	} else {
		salt = append([]byte(nil), salt...)
	}

	key := pbkdf2.Key([]byte(password), salt, iterations, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, salt, nil
}

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
