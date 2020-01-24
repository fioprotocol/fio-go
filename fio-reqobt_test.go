package fio

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/eoscanada/eos-go"
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

func TestDecode(t *testing.T) {
	obt := ObtRequestContent{
		PayeePublicAddress: "purse.alice",
		Amount:             "1",
		TokenCode:          "fio.reqobt",
	}
	// our encode and theirs will probably be different, so encode to json for compare
	const newFundsContentHex = "0B70757273652E616C69636501310A66696F2E7265716F6274000000"
	obtBin, _ := hex.DecodeString(newFundsContentHex)
	abiReader := bytes.NewReader([]byte(ObtAbiJson))
	abi, err := eos.NewABI(abiReader)
	if err != nil {
		t.Error(err)
	}
	obt2J, err := abi.DecodeTableRowTyped("new_funds_content", obtBin)
	if err != nil {
		t.Error(err)
	}
	obt2 := &ObtRequestContent{}
	json.Unmarshal(obt2J, obt2)
	j2, _ := json.MarshalIndent(obt2, "", "  ")
	j, _ := json.MarshalIndent(obt, "", "  ")
	if string(j) != string(j2) {
		fmt.Println(string(j))
		fmt.Println(string(j2))
		t.Error("content didn't match on encode and decode")
	}

	//iv, _ := hex.DecodeString("f300888ca4f512cebdc0020ff0f7224c")
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
		//req := ObtRecordContent{PayerPublicAddress: hex.EncodeToString(someData)}

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

func TestEciesDecrypt(t *testing.T) {
	iv, _ := hex.DecodeString("f300888ca4f512cebdc0020ff0f7224c")
	alice, _ := NewAccountFromWif("5J9bWm2ThenDm3tjvmUgHtWCVMUdjRR1pxnRtnJjvKA4b2ut5WK")
	bob, _ := NewAccountFromWif("5JoQtsKQuH8hC9MyvfJAqo6qmKLm8ePYNucs7tPu2YxG12trzBt")
	knownSecret := "a71b4ec5a9577926a1d2aa1d9d99327fd3b68f6a1ea597200a0d890bd3331df300a2d49fec0b2b3e6969ce9263c5d6cf47c191c1ef149373ecc9f0d98116b598"
	cipherText := "f300888ca4f512cebdc0020ff0f7224c0db2984c4ad9afb12629f01a8c6a76328bbde17405655dc4e3cb30dad272996fb1dea8e662e640be193e25d41147a904c571b664a7381ab41ef062448ac1e205"
	knownEncoded := "0b70757273652e616c69636501310a66696f2e7265716f6274000000"

	//mySecretSeed, mySecret, _ := EciesSecret(alice, bob.PubKey)
	_, mySecret, _ := EciesSecret(alice, bob.PubKey)
	if knownSecret != hex.EncodeToString(mySecret[:]) {
		fmt.Println("---- alice -> bob secret vs known value")
		fmt.Println("expected", knownSecret)
		fmt.Println("mine    ", hex.EncodeToString(mySecret[:]))
		t.Error("secret value did not match")
	} else {
		fmt.Println("ecies secret derivation matched alice -> bob.")
	}

	mySecretSeed, mySecret, _ := EciesSecret(bob, alice.PubKey)
	if knownSecret != hex.EncodeToString(mySecret[:]) {
		fmt.Println("---- bob -> alice secret vs known value")
		fmt.Println("expected", knownSecret)
		fmt.Println("mine    ", hex.EncodeToString(mySecret[:]))
		t.Error("secret value did not match")
	} else {
		fmt.Println("ecies secret derivation matched bob -> alice.")
	}
	if mySecretSeed != nil {
		fmt.Println("secret seed value", mySecretSeed)
	}

	obtRequest := ObtRequestContent{
		PayeePublicAddress: "purse.alice",
		Amount:             "1",
		TokenCode:          "fio.reqobt",
		Memo:               "",
		Hash:               "",
		OfflineUrl:         "",
	}
	j, _ := json.Marshal(&obtRequest)
	abiReader := bytes.NewReader([]byte(ObtAbiJson))
	abi, _ := eos.NewABI(abiReader)
	b, err := abi.EncodeAction("new_funds_content", j)
	if err != nil {
		t.Error(err)
	}
	if knownEncoded != hex.EncodeToString(b) {
		fmt.Println("expected:", knownEncoded)
		fmt.Println("mine:    ", hex.EncodeToString(b))
		t.Error("struct did not match when abi encoded.")
	} else {
		fmt.Println("abi encoding matched for new_funds_content")
	}

	myCipherText, err := EciesEncrypt(bob, alice.PubKey, b, iv)
	if err != nil {
		t.Error(err)
	}
	if myCipherText != cipherText {
		fmt.Println("AES encryption did not match.")
		fmt.Println("expected", cipherText[:32], cipherText[32:len(cipherText)-64], cipherText[64:])
		fmt.Println("mine    ", myCipherText[:32], myCipherText[32:len(myCipherText)-64], myCipherText[64:])
		t.Error("ciphertext does not match given a fixed iv")
	} else {
		fmt.Println("known message encrypted to same value.")
	}

}
