package cowyo

import (
	"errors"
	"strings"
	"testing"
)

func TestPageLockCredentialsRoundTrip(t *testing.T) {
	const password = "correct horse battery staple"

	credentials, err := createPageLock(password)
	if err != nil {
		t.Fatalf("createPageLock() error = %v", err)
	}
	if credentials.salt == "" || credentials.verifier == "" {
		t.Fatalf("createPageLock() = %+v, want populated credentials", credentials)
	}
	if credentials.salt == password || credentials.verifier == password {
		t.Fatal("page lock credentials contain the plaintext password")
	}
	if err := verifyPageLock(credentials, password); err != nil {
		t.Fatalf("verifyPageLock() error = %v", err)
	}
}

func TestPageLockRejectsWrongPasswordAndInvalidCredentials(t *testing.T) {
	credentials, err := createPageLock("correct password")
	if err != nil {
		t.Fatalf("createPageLock() error = %v", err)
	}

	if err := verifyPageLock(credentials, "incorrect password"); !errors.Is(err, errWrongLockPassword) {
		t.Fatalf("wrong-password error = %v, want %v", err, errWrongLockPassword)
	}

	invalidSalt := credentials
	invalidSalt.salt = "***"
	if err := verifyPageLock(invalidSalt, "correct password"); err == nil {
		t.Fatal("verifyPageLock() accepted an invalid salt")
	}

	invalidVerifier := credentials
	invalidVerifier.verifier = "***"
	if err := verifyPageLock(invalidVerifier, "correct password"); err == nil {
		t.Fatal("verifyPageLock() accepted an invalid verifier")
	}
}

func TestPageLockRequiresStrongPasswordAndUsesRandomSalts(t *testing.T) {
	if _, err := createPageLock("short"); err == nil {
		t.Fatal("createPageLock() accepted a short password")
	}
	tooLong := strings.Repeat("a", maxLockPasswordLen+1)
	if _, err := createPageLock(tooLong); err == nil {
		t.Fatal("createPageLock() accepted an oversized password")
	}

	first, err := createPageLock("long enough password")
	if err != nil {
		t.Fatalf("first createPageLock() error = %v", err)
	}
	second, err := createPageLock("long enough password")
	if err != nil {
		t.Fatalf("second createPageLock() error = %v", err)
	}
	if first == second {
		t.Fatal("createPageLock() reused the same salt and verifier")
	}
}
