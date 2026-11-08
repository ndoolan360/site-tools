// Package encryption encrypts page content for client-side decryption.
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"

	sitetools "go.doolan.dev/site-tools"
)

//go:embed assets/encryption_decrypt_script.js
var decryptScriptTemplate []byte

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
// AESGCMEncryption or the optional encryption/age package). The fields on the transformer itself
// only configure the surrounding template wiring.
type EncryptionTransformer struct {
	Template *sitetools.Asset
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
	// ScriptTransformer optionally transforms the rendered decryption JavaScript.
	// It receives an asset with path "encryption_decrypt_script.js" after placeholder
	// replacement. Nil leaves the script unchanged. Algorithm preambles are not transformed.
	ScriptTransformer sitetools.Transformer
}

var _ sitetools.Transformer = EncryptionTransformer{}
var _ EncryptionAlgorithm = (*AESGCMEncryption)(nil)

func (t EncryptionTransformer) Transform(asset *sitetools.Asset) error {
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

	decryptionScript, err := renderDecryptionScript(t.Algorithm.ScriptTemplate(), replacements, t.ScriptTransformer)
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
// optionally transforms the result.
func renderDecryptionScript(template []byte, replacements map[string]string, transformer sitetools.Transformer) (string, error) {
	scriptAsset := &sitetools.Asset{
		Path: "encryption_decrypt_script.js",
		Data: template,
	}
	if err := (sitetools.ReplacerTransformer{Replacements: replacements}).Transform(scriptAsset); err != nil {
		return "", err
	}
	if transformer != nil {
		if err := transformer.Transform(scriptAsset); err != nil {
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

	key, err := pbkdf2.Key(sha256.New, password, salt, iterations, 32)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to derive AES-GCM key: %w", err)
	}

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
