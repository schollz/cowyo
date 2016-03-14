package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
)

var encryptionType string

func init() {
	encryptionType = "PGP SIGNATURE"
}

func encryptString(encryptionText string, encryptionPassphraseString string) string {
	encryptionPassphrase := []byte(encryptionPassphraseString)
	encbuf := bytes.NewBuffer(nil)
	w, err := armor.Encode(encbuf, encryptionType, nil)
	if err != nil {
		log.Fatal(err)
	}

	plaintext, err := openpgp.SymmetricallyEncrypt(w, encryptionPassphrase, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	message := []byte(encryptionText)
	_, err = plaintext.Write(message)

	plaintext.Close()
	w.Close()
	return encbuf.String()
}

func decryptString(decryptionString string, encryptionPassphraseString string) (string, error) {
	encryptionPassphrase := []byte(encryptionPassphraseString)
	decbuf := bytes.NewBuffer([]byte(decryptionString))
	result, err := armor.Decode(decbuf)
	if err != nil {
		return "", err
	}

	alreadyPrompted := false
	md, err := openpgp.ReadMessage(result.Body, nil, func(keys []openpgp.Key, symmetric bool) ([]byte, error) {
		if alreadyPrompted {
			return nil, errors.New("Could not decrypt using passphrase")
		} else {
			alreadyPrompted = true
		}
		return encryptionPassphrase, nil
	}, nil)
	if err != nil {
		return "", err
	}

	bytes, err := ioutil.ReadAll(md.UnverifiedBody)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// func main() {
// 	test := encryptString("This is some string", "golang")
// 	fmt.Println(test)
// 	testD := decryptString(test, "golang")
// 	fmt.Println(testD)
//
// }
