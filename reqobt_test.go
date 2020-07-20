package fio

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestOBT(t *testing.T) {
	alice, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}

	bob, err := NewAccountFromWif(`5KQ6f9ZgUtagD3LZ4wcMKhhvK9qy4BuwL3L1pkm6E2v62HCne2R`)
	if err != nil {
		t.Error(err)
		return
	}
	apiB, _, err := NewConnection(bob.KeyBag, api.BaseURL)
	if err != nil {
		t.Error(err)
		return
	}
	aliceAddresses, err := api.GetFioAddresses(alice.PubKey, 0, 100)
	if err != nil {
		t.Error(err)
	}
	bobAddresses, err := api.GetFioAddresses(bob.PubKey, 0, 100)
	if err != nil {
		t.Error(err)
	}

	// already have it, but be thorough and walk through entire process
	bobPub, found, err := api.PubAddressLookup(Address(bobAddresses.FioAddresses[0].FioAddress), "FIO", "FIO")
	if err != nil || !found {
		t.Error("can't get pubaddress, giving up ", err)
		return
	}

	// encrypt 3 requests
	requests := make([]string, 3)
	for i := 1; i <= 3; i++ {
		requests[i-1], err = ObtRequestContent{
			PayeePublicAddress: alice.PubKey,
			Amount:             fmt.Sprintf("%d", i),
			ChainCode:          "FIO",
			TokenCode:          "FIO",
			Memo:               fmt.Sprintf("request %d", i),
		}.Encrypt(alice, bobPub.PublicAddress)
		if err != nil {
			t.Error(err)
		}
	}

	// alice sends them to bob
	for _, r := range requests {
		_, err := api.SignPushActions(
			NewFundsReq(alice.Actor, bobAddresses.FioAddresses[0].FioAddress, aliceAddresses.FioAddresses[0].FioAddress, r),
		)
		if err != nil {
			t.Error(err)
			return
		}
		time.Sleep(500 * time.Millisecond)
	}

	// check if we have sent requests, and cancel the last one
	sent, ok, err := api.GetSentFioRequests(alice.PubKey, 100, 0)
	if err != nil {
		t.Error(err)
		return
	}
	if !ok {
		t.Error("no sent requests")
		return
	}
	cnlReq := NewCancelFndReq(alice.Actor, sent.Requests[len(sent.Requests)-1].FioRequestId)
	_, err = api.SignPushActions(
		cnlReq,
	)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(250 * time.Millisecond)
	// ensure it's on the list of cancelled requests
	cancelled, err := api.GetCancelledRequests(alice.PubKey, 100, 0)
	if err != nil {
		t.Error(err)
	} else if cancelled.Requests == nil || len(cancelled.Requests) == 0 {
		t.Error("did not have any cancelled requests")
	} else {
		if cancelled.Requests[len(cancelled.Requests)-1].FioRequestId != sent.Requests[len(sent.Requests)-1].FioRequestId {
			t.Error("did not find cancelled request")
		}
	}

	// now bob's turn, get pending requests
	pending, ok, err := apiB.GetPendingFioRequests(bob.PubKey, 100, 0)
	if err != nil {
		t.Error(err)
		return
	}
	if !ok {
		t.Error("no pending requests")
		return
	}

	// find the last one from alice, ensure it's request 2, then reject
	for i := len(pending.Requests) - 1; i >= 0; i-- {
		if pending.Requests[i].PayeeFioPublicKey == alice.PubKey {
			fndReq, err := DecryptContent(bob, alice.PubKey, pending.Requests[i].Content, ObtRequestType)
			if err != nil {
				t.Error(err)
				break
			}
			if fndReq.Request.Amount != "2" || fndReq.Request.Memo != "request 2" {
				t.Error("fund request did not have expected content")
				if j, err := fndReq.ToJson(); err == nil {
					fmt.Println(string(j))
				}
				break
			}
			_, err = apiB.SignPushActions(
				NewRejectFndReq(bob.Actor, fmt.Sprintf("%d", pending.Requests[i].FioRequestId)),
			)
			if err != nil {
				t.Error(err)
				break
			}
			break
		}
	}

	// ensure we have one less pending request
	afterRej, ok, err := apiB.GetPendingFioRequests(bob.PubKey, 100, 0)
	if err != nil {
		t.Error(err)
		return
	}
	if !ok {
		t.Error("no pending requests")
		return
	}
	if len(pending.Requests)-1 != len(afterRej.Requests) {
		t.Error("rejecting fund request did not remove from pending list")
	}

	// finally record a response to the remaining request
	for i := len(afterRej.Requests) - 1; i >= 0; i-- {
		if pending.Requests[i].PayeeFioPublicKey == alice.PubKey {
			fndReq, err := DecryptContent(bob, alice.PubKey, pending.Requests[i].Content, ObtRequestType)
			if err != nil {
				t.Error(err)
				break
			}
			if fndReq.Request.Amount != "1" || fndReq.Request.Memo != "request 1" {
				t.Error("fund request did not have expected content")
				if j, err := fndReq.ToJson(); err == nil {
					fmt.Println(string(j))
				}
				break
			}
			content, err := ObtRecordContent{
				PayerPublicAddress: bob.PubKey,
				PayeePublicAddress: alice.PubKey,
				Amount:             "1",
				ChainCode:          "FIO",
				TokenCode:          "FIO",
				ObtId:              "here is your money",
			}.Encrypt(bob, alice.PubKey)
			if err != nil {
				t.Error(err)
				break
			}
			_, err = apiB.SignPushActions(
				NewRecordSend(
					bob.Actor,
					fmt.Sprintf("%d", afterRej.Requests[i].FioRequestId),
					bobAddresses.FioAddresses[0].FioAddress,
					aliceAddresses.FioAddresses[0].FioAddress,
					content,
				),
			)
			if err != nil {
				t.Error(err)
				break
			}
			break
		}
	}

}

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
	if !bytes.Equal(a[:], b[:]) {
		t.Error("dh-ecdsa secret did not match")
		return
	}
	// the example only gives the first 50 bytes, but that's good enough.
	known, _ := hex.DecodeString(`a71b4ec5a9577926a1d2aa1d9d99327fd3b68f6a1ea597200a0d890bd3331df300a2d49fec0b2b3e6969ce9263c5d6cf47c1`)
	if !bytes.Equal(known, a[:len(known)]) {
		t.Error("secret did not decode to known good value")
	}
}

func TestEciesSecret2(t *testing.T) {
	const expectCipherText = "f300888ca4f512cebdc0020ff0f7224c0db2984c4ad9afb12629f01a8c6a76328bbde17405655dc4e3cb30dad272996fb1dea8e662e640be193e25d41147a904c571b664a7381ab41ef062448ac1e205"
	// hard coding values from typescript unit tests to ensure same result ...

	// Check shared-secret derivation
	aWif := "5J9bWm2ThenDm3tjvmUgHtWCVMUdjRR1pxnRtnJjvKA4b2ut5WK"
	bWif := "5JoQtsKQuH8hC9MyvfJAqo6qmKLm8ePYNucs7tPu2YxG12trzBt"
	alice, _ := NewAccountFromWif(aWif)
	bob, _ := NewAccountFromWif(bWif)
	expectSecret := []byte{167, 27, 78, 197, 169, 87, 121, 38, 161, 210, 170, 29, 157, 153, 50, 127, 211, 182, 143, 106, 30, 165, 151, 32, 10, 13, 137, 11, 211, 51, 29, 243, 0, 162, 212, 159, 236, 11, 43, 62, 105, 105, 206, 146, 99, 197, 214, 207, 71, 193, 145, 193, 239, 20, 147, 115, 236, 201, 240, 217, 129, 22, 181, 152}
	_, secretHash, err := EciesSecret(alice, bob.PubKey)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(expectSecret, secretHash[:]) {
		t.Error("derived secret does not match expected (hard-coded) value")
	}
	_, secretHash2, err := EciesSecret(bob, alice.PubKey)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(secretHash[:], secretHash2[:]) {
		t.Error("bob and alice didn't agree on a shared secret")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	// run through it several times with random data, keys, and length to ensure padding, etc works.
	for i := 0; i < 40; i++ {
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

		// test the encrypt/decrypt on raw bytes first
		cipherText, e := EciesEncrypt(sender, recipient.PubKey, someData, nil)
		if e != nil {
			t.Error(e.Error())
			return
		}
		decrypted, e := EciesDecrypt(recipient, sender.PubKey, cipherText)
		if e != nil {
			t.Error(e.Error())
			return
		}
		if hex.EncodeToString(someData) != hex.EncodeToString(decrypted) {
			fmt.Println(hex.EncodeToString(someData))
			fmt.Println(hex.EncodeToString(decrypted))
			t.Error("decrypted content from EciesEncrypt did not match EciesDecrypt output")
			return
		}

		// now do it again vs an ObtContent struct.
		req := ObtRecordContent{
			PayerPublicAddress: "aaaaaaaaaa",
			PayeePublicAddress: "bbbbbbbbbb",
			Amount:             "1111111111",
			TokenCode:          "zzzzzzzzzz",
			Status:             "xxxxxxxxxx",
			ObtId:              "2222222222",
			Memo:               "ffffffffff",
		}

		content, e := req.Encrypt(sender, recipient.PubKey)
		if e != nil {
			t.Error(e.Error())
			return
		} else if len(content) == 0 {
			t.Error("got empty result for encrypted data")
			return
		}

		resp, e := DecryptContent(recipient, sender.PubKey, content, ObtResponseType)
		if e != nil {
			t.Error(e.Error())
			return
		} else if resp == nil {
			t.Error("resp is nil")
			return
		}
		if req.PayerPublicAddress != resp.Record.PayerPublicAddress {
			t.Error("decrypted content does not match")
			return
		}
	}
}
