package fio

import (
	"bytes"
	"math/rand"
	"testing"
	"time"
)

func TestEncryptDecrypt(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	// run through it several times with random data, and length to ensure padding, etc works.
	for i:= 0; i<10; i++ {
		size := rand.Intn(128) + 128
		someData := make([]byte, size)
		_, e := rand.Read(someData)
		if e != nil {
			t.Error(e.Error())
			return
		}
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
		cipherText, e := EncryptContent(sender, recipient.PubKey, someData)
		if e != nil {
			t.Error(e.Error())
			return
		}
		decrypted, e := DecryptContent(recipient, sender.PubKey, cipherText)
		if e != nil {
			t.Error(e.Error())
			return
		}
		if !bytes.Equal(someData, decrypted) {
			t.Error("decrypted content from EncryptContent did not match DecryptContent output")
		}
	}
}
