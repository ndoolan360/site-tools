package sitetools

import (
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

	"golang.org/x/crypto/pbkdf2"
)

//go:embed encryption_decrypt_script.js
var decryptScriptTemplate []byte

type StorageMode string

const (
	StoreNone    StorageMode = "noOpStorage"
	StoreLocal   StorageMode = "window.localStorage"
	StoreSession StorageMode = "window.sessionStorage"
)

// EncryptionTransformer encrypts page content with client-side decryption.
type EncryptionTransformer struct {
	Template *Asset
	// Password is used to encrypt the content. It is NOT stored in the output - only used during encryption.
	Password string
	// Iterations is the number of PBKDF2 iterations for key derivation (default: 600000).
	// Higher values increase security but also increase decryption time in the browser.
	Iterations int
	// Salt is optional. If provided, it is reused for all pages encrypted by this transformer.
	// If empty, a random salt is generated per page (default behavior).
	// Setting this to a fixed value is less secure but allows for faster decryption, consistent
	// encrypted output across builds, and sharing the same password across multiple pages if they
	// also share the same iterations value.
	Salt []byte
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

	if t.Password == "" {
		return fmt.Errorf("password is required for encryption")
	}

	passwordInputID := t.PasswordInputID
	formID := t.FormID
	contentID := t.ContentID
	iterations := t.Iterations
	storageMode := t.StorageMode

	// Set default IDs if not provided
	if passwordInputID == "" {
		passwordInputID = "password"
	}
	if formID == "" {
		formID = "password-form"
	}
	if contentID == "" {
		contentID = "encrypted-content"
	}
	if iterations == 0 {
		iterations = 600000
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

	// Encrypt the asset data
	encryptedData, salt, err := encrypt(asset.Data, t.Password, iterations, t.Salt)
	if err != nil {
		return fmt.Errorf("failed to encrypt asset: %w", err)
	}

	// Encode encrypted data and salt as base64
	encryptedBase64 := base64.StdEncoding.EncodeToString(encryptedData)
	saltBase64 := base64.StdEncoding.EncodeToString(salt)

	// Generate the decryption JavaScript
	decryptionScript, err := generateDecryptionScript(
		encryptedBase64,
		saltBase64,
		iterations,
		passwordInputID,
		formID,
		contentID,
		storageMode,
		t.MinifyScript,
	)
	if err != nil {
		return fmt.Errorf("failed to generate decryption script: %w", err)
	}

	// Wrap main script in script tags
	scriptTag := "\n<script>\n" + decryptionScript + "\n</script>\n"

	// Insert script into the template (before closing body)
	var finalHTML string
	if closeBodyIdx := strings.LastIndex(templateStr, "</body>"); closeBodyIdx != -1 {
		finalHTML = templateStr[:closeBodyIdx] + scriptTag + templateStr[closeBodyIdx:]
	} else if closeHTMLIdx := strings.LastIndex(templateStr, "</html>"); closeHTMLIdx != -1 {
		finalHTML = templateStr[:closeHTMLIdx] + scriptTag + templateStr[closeHTMLIdx:]
	} else {
		finalHTML = templateStr + scriptTag
	}

	asset.Data = []byte(finalHTML)

	return nil
}

// RandomSalt returns a 32-byte random salt suitable for PBKDF2.
// Use this if you want to supply a fixed salt for an EncryptionTransformer.
func RandomSalt() ([]byte, error) {
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// encrypt encrypts data using AES-GCM with PBKDF2 key derivation
func encrypt(data []byte, password string, iterations int, salt []byte) ([]byte, []byte, error) {
	if len(salt) == 0 {
		var err error
		salt, err = RandomSalt()
		if err != nil {
			return nil, nil, err
		}
	} else {
		salt = append([]byte(nil), salt...)
	}

	// Derive key using PBKDF2
	key := pbkdf2.Key([]byte(password), salt, iterations, 32, sha256.New)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	// Encrypt and authenticate
	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	return ciphertext, salt, nil
}

// generateDecryptionScript creates the JavaScript code for client-side decryption
func generateDecryptionScript(
	encryptedData, salt string,
	iterations int,
	passwordInputID, formID, contentID string,
	storageMode StorageMode,
	minifyScript bool,
) (string, error) {
	// Create an asset for the script template
	scriptAsset := &Asset{
		Path: "encryption_template.js",
		Data: decryptScriptTemplate,
	}

	// Use ReplacerTransformer to replace placeholders
	err := ReplacerTransformer{
		Replacements: map[string]string{
			"{{.ENCRYPTED_DATA}}":        encryptedData,
			"{{.ENCRYPTED_DATA_PREFIX}}": encryptedData[:32],
			"{{.SALT}}":                  salt,
			"{{.ITERATIONS}}":            strconv.Itoa(iterations),
			"{{.PASSWORD_INPUT_ID}}":     passwordInputID,
			"{{.FORM_ID}}":               formID,
			"{{.CONTENT_ID}}":            contentID,
			"{{.STORAGE_MODE}}":          string(storageMode),
		},
	}.Transform(scriptAsset)
	if err != nil {
		return "", err
	}

	if minifyScript {
		err := MinifyTransformer{}.Transform(scriptAsset)
		if err != nil {
			return "", err
		}
	}

	return string(scriptAsset.Data), nil
}
