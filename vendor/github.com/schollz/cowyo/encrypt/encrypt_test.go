package encrypt

import "testing"

func TestEncryption(t *testing.T) {
	s, err := EncryptString("some string", "some password")
	if err != nil {
		t.Errorf("What")
	}
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
