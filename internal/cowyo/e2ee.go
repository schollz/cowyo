package cowyo

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

const (
	e2eeDocumentStart = "-----BEGIN COWYO E2EE DOCUMENT V1-----"
	e2eeDocumentEnd   = "-----END COWYO E2EE DOCUMENT V1-----"
)

var (
	errE2EEAuthenticationRequired = errors.New("private page authentication required")
	errE2EEInvalidCapability      = errors.New("invalid private page write capability")
	errE2EEInvalidDocument        = errors.New("invalid COWYO E2EE DOCUMENT V1 envelope")
	errE2EEPageCollision          = errors.New("private page name is already in use")
	errE2EEConversionUnavailable  = errors.New("page cannot be converted to private mode")
	errE2EEIncompatibleOperation  = errors.New("operation is unavailable for private pages")
)

var e2eeDocumentPattern = regexp.MustCompile(
	`\A` + regexp.QuoteMeta(e2eeDocumentStart) +
		`\r?\n([^\r\n]+)\r?\n` +
		regexp.QuoteMeta(e2eeDocumentEnd) + `\z`,
)

type e2eeDocumentEnvelope struct {
	Version    int    `json:"v"`
	Cipher     string `json:"cipher"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"data"`
}

func validE2EEDocument(text string) bool {
	match := e2eeDocumentPattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return false
	}

	var envelope e2eeDocumentEnvelope
	decoder := json.NewDecoder(strings.NewReader(match[1]))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return false
	}
	if envelope.Version != 1 || envelope.Cipher != "xchacha20-poly1305" {
		return false
	}

	nonce, err := decodeStrictBase64URL(envelope.Nonce, 24)
	if err != nil || len(nonce) != 24 {
		return false
	}
	ciphertext, err := decodeStrictBase64URL(envelope.Ciphertext, -1)
	return err == nil && len(ciphertext) >= 16
}

func e2eeCapabilityHash(capability string) (string, error) {
	decoded, err := decodeStrictBase64URL(capability, sha256.Size)
	if err != nil {
		return "", errE2EEInvalidCapability
	}
	hash := sha256.Sum256(decoded)
	return base64.RawURLEncoding.EncodeToString(hash[:]), nil
}

func verifyE2EECapability(storedHash, capability string) error {
	want, err := decodeStrictBase64URL(storedHash, sha256.Size)
	if err != nil {
		return errE2EEInvalidCapability
	}
	capabilityHash, err := e2eeCapabilityHash(capability)
	if err != nil {
		return err
	}
	got, err := decodeStrictBase64URL(capabilityHash, sha256.Size)
	if err != nil || subtle.ConstantTimeCompare(want, got) != 1 {
		return errE2EEInvalidCapability
	}
	return nil
}

func decodeStrictBase64URL(value string, size int) ([]byte, error) {
	if value == "" || strings.Contains(value, "=") {
		return nil, fmt.Errorf("base64url value is empty or padded")
	}
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil || base64.RawURLEncoding.EncodeToString(decoded) != value {
		return nil, fmt.Errorf("invalid base64url value")
	}
	if size >= 0 && len(decoded) != size {
		return nil, fmt.Errorf("base64url value has invalid length")
	}
	return decoded, nil
}
