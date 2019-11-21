package fio

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	plainText := []byte(`this is a test`)
	sender, e := NewRandomAccount()
	if e != nil {
		t.Error(e.Error())
		return
	}
	recipient, e := NewRandomAccount()
	if e != nil {
		t.Error(e.Error())
		return
	}
	cipherText, e := EncryptContent(sender, recipient.PubKey, plainText)
	if e != nil {
		t.Error(e.Error())
		return
	}
	decrypted, e := DecryptContent(recipient, sender.PubKey, cipherText)
	if e != nil {
		t.Error(e.Error())
		return
	}
	if !bytes.Equal(plainText, decrypted) {
		t.Error("decrypted content from EncryptContent did not match DecryptContent output")
	}
}
