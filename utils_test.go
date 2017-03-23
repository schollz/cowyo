package main

import (
	"testing"
)

func TestHashing(t *testing.T) {
	p := HashPassword("1234")
	log.Debug(p)
	err := CheckPasswordHash("1234", p)
	if err != nil {
		t.Errorf("Should be correct password")
	}
	err = CheckPasswordHash("1234lkjklj", p)
	if err == nil {
		t.Errorf("Should NOT be correct password")
	}
}

func TestEncryption(t *testing.T) {
	s, err := EncryptString("some string", "some password")
	if err != nil {
		t.Errorf("What")
	}
	log.Debug(s)
	d, err := DecryptString(s, "some wrong password")
	if err == nil {
		t.Errorf("Should throw error for bad password")
	}
	d, err = DecryptString(s, "some password")
	if err != nil {
		t.Errorf("Should not throw password")
	}
	if d != "some string" {
		t.Errorf("Problem decoding")
	}

}
