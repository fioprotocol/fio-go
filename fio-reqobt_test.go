package fio

import (
	"bytes"
	"encoding/hex"
	"math/rand"
	"testing"
	"time"
)

func TestEciesSecret(t *testing.T) {
	// test is based on the example in the fiojs package, and ensures we get the same secret ...
	// https://github.com/fioprotocol/fiojs/blob/master/docs/message_encryption.md
	bob, _ := NewAccountFromWif(`5JoQtsKQuH8hC9MyvfJAqo6qmKLm8ePYNucs7tPu2YxG12trzBt`)
	alice, _ := NewAccountFromWif(`5J9bWm2ThenDm3tjvmUgHtWCVMUdjRR1pxnRtnJjvKA4b2ut5WK`)

	_, a, e := EciesSecret(bob, alice.PubKey)
	if e != nil {
		t.Error(e.Error())
	}
	_, b, e := EciesSecret(alice, bob.PubKey)
	if e != nil {
		t.Error(e.Error())
	}
	if !bytes.Equal(a, b) {
		t.Error("dh-ecdsa secret did not match")
		return
	}
	// the example only gives the first 50 bytes, but that's good enough.
	known, _ := hex.DecodeString(`a71b4ec5a9577926a1d2aa1d9d99327fd3b68f6a1ea597200a0d890bd3331df300a2d49fec0b2b3e6969ce9263c5d6cf47c1`)
	if !bytes.Equal(known, a[:len(known)]) {
		t.Error("secret did not decode to known good value")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	// run through it several times with random data, keys, and length to ensure padding, etc works.
	for i := 0; i < 10; i++ {
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
