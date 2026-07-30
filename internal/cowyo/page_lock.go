package cowyo

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/scrypt"
)

const (
	pageLockN          = 1 << 16
	pageLockR          = 8
	pageLockP          = 1
	pageLockSaltLength = 16
	pageLockKeyLength  = 32
	minLockPasswordLen = 8
	maxLockPasswordLen = 1024
)

var (
	errPageAlreadyLocked = errors.New("page is already locked")
	errPageNotLocked     = errors.New("page is not locked")
	errWrongLockPassword = errors.New("wrong page lock password")
)

type pageLockCredentials struct {
	salt     string
	verifier string
}

func createPageLock(password string) (pageLockCredentials, error) {
	if len(password) < minLockPasswordLen {
		return pageLockCredentials{}, fmt.Errorf(
			"page lock password must be at least %d characters",
			minLockPasswordLen,
		)
	}
	if len(password) > maxLockPasswordLen {
		return pageLockCredentials{}, fmt.Errorf(
			"page lock password must not exceed %d bytes",
			maxLockPasswordLen,
		)
	}

	salt := make([]byte, pageLockSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return pageLockCredentials{}, fmt.Errorf("generate page lock salt: %w", err)
	}
	verifier, err := scrypt.Key(
		[]byte(password),
		salt,
		pageLockN,
		pageLockR,
		pageLockP,
		pageLockKeyLength,
	)
	if err != nil {
		return pageLockCredentials{}, fmt.Errorf("derive page lock verifier: %w", err)
	}

	return pageLockCredentials{
		salt:     base64.RawURLEncoding.EncodeToString(salt),
		verifier: base64.RawURLEncoding.EncodeToString(verifier),
	}, nil
}

func verifyPageLock(credentials pageLockCredentials, password string) error {
	salt, err := base64.RawURLEncoding.DecodeString(credentials.salt)
	if err != nil || len(salt) != pageLockSaltLength {
		return errors.New("stored page lock salt is invalid")
	}
	expected, err := base64.RawURLEncoding.DecodeString(credentials.verifier)
	if err != nil || len(expected) != pageLockKeyLength {
		return errors.New("stored page lock verifier is invalid")
	}

	actual, err := scrypt.Key(
		[]byte(password),
		salt,
		pageLockN,
		pageLockR,
		pageLockP,
		pageLockKeyLength,
	)
	if err != nil {
		return fmt.Errorf("derive page lock verifier: %w", err)
	}
	if subtle.ConstantTimeCompare(actual, expected) != 1 {
		return errWrongLockPassword
	}
	return nil
}
