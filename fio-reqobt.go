package fio

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"github.com/eoscanada/eos-go/btcsuite/btcutil"
	"github.com/eoscanada/eos-go/ecc"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"math/rand"
	"time"
)

type ObtContent struct {
	PayerPublicAddress string `json:"payer_public_address"`
	PayeePublicAddress string `json:"payee_public_address"`
	Amount             string `json:"amount"`
	TokenCode          string `json:"token_code"`
	Status             string `json:"status"`
	ObtId              string `json:"obt_id"`
	Memo               string `json:"memo"`
	Hash               string `json:"hash"`
	OfflineUrl         string `json:"offline_url"`
}

// TODO: need to figure out how to encrypt the Content field, no use building further until that works ...
type RecordSend struct {
	FioRequestId    string `json:"fio_request_id"`
	PayerFioAddress string `json:"payer_fio_address"`
	PayeeFioAddress string `json:"payee_fio_address"`
	Content         string `json:"content"`
	MaxFee          uint64 `json:"max_fee"`
	Actor           string `json:"actor"` // NOTE this differs from other fio.* contracts, and is a string not name!!!
	Tpid            string `json:"tpid"`
}

type NewFundsReq struct {
	PayerFioAddress string `json:"payer_fio_address"`
	PayeeFioAddress string `json:"payee_fio_address"`
	Content         string `json:"content"`
	MaxFee          uint64 `json:"max_fee"`
	Actor           string `json:"actor"`
	Tpid            string `json:"tpid"`
}

type RejectFndReq struct {
	FioRequestId string `json:"fio_request_id"`
	MaxFee       uint64 `json:"max_fee"`
	Actor        string `json:"actor"`
	Tpid         string `json:"tpid"`
}

// EncryptContent implements the encryption format used in the content field of OBT requests. A DH shared secret is
// created using ECIES which derives a shared secret based on the curves of the public and private keys.
// This secret is hashed using sha512, and the first 32 bytes of the hash is used to encrypt the message using
// AES-256 cbc, and the second half is used to create an outer sha256 hmac. A 16 byte IV is prepended to the
// output, resulting in the message format of: IV + Ciphertext + HMAC
// See https://github.com/fioprotocol/fiojs/blob/master/docs/message_encryption.md for more information.
func EncryptContent(sender *Account, recipentPub string, plainText []byte) (content []byte, err error) {
	var buffer bytes.Buffer

	// convert to sender key to ecies private key type
	wif, err := btcutil.DecodeWIF(sender.KeyBag.Keys[0].String())
	if err != nil {
		return nil, err
	}
	priv := ecies.ImportECDSA(wif.PrivKey.ToECDSA())

	// convert recipient public key to ecies private key type
	eosPub, e := ecc.NewPublicKey(`EOS` + recipentPub[3:])
	if e != nil {
		return nil, e
	}
	epk, err := eosPub.Key()
	if err != nil {
		return nil, err
	}
	pub := ecies.ImportECDSAPublic(epk.ToECDSA())

	// generate the shared key, and hash it
	sharedKey, err := priv.GenerateShared(pub,32, 0)
	if err != nil {
		return nil, err
	}
	secretHash := sha512.New().Sum(sharedKey)

	// Generate IV
	iv := make([]byte, 16)
	rand.Seed(time.Now().UnixNano())
	_, e = rand.Read(iv)
	if e != nil {
		return nil, e
	}
	buffer.Write(iv)

	// setup AES CBC for encryption
	block, e := aes.NewCipher(secretHash[:32])
	if e != nil {
		return nil, e
	}
	cbc := cipher.NewCBCEncrypter(block, iv)

	// create pkcs#7 padding
	pad := func() []byte {
		padLen := block.BlockSize() - (len(plainText) % block.BlockSize())
		if padLen == 0 {
			padLen = block.BlockSize()
		}
		pad := make([]byte, 0)
		for i := 0; i < padLen; i++ {
			pad = append(pad, byte(padLen))
		}
		return pad
	}()

	// encrypt the plaintext
	cipherText := make([]byte, len(plainText) + len(pad))
	cbc.CryptBlocks(cipherText, append(plainText, pad...))
	buffer.Write(cipherText)

	// Sign the message using sha256 hmac
	signer := hmac.New(sha256.New, secretHash[32:])
	signature := signer.Sum(buffer.Bytes())
	buffer.Write(signature)

	return buffer.Bytes(), nil
}

// DecryptContent is the inverse of EncryptContent, using the recipient's private key and sender's public instead.
func DecryptContent(recipient *Account, senderPub string, message []byte) (decrypted []byte, err error) {
	// convert recipient private to ecies private key type
	wif, err := btcutil.DecodeWIF(recipient.KeyBag.Keys[0].String())
	if err != nil {
		return nil, err
	}
	priv := ecies.ImportECDSA(wif.PrivKey.ToECDSA())

	// convert sender into an ecies public key struct
	eosPub, err := ecc.NewPublicKey(`EOS` + senderPub[3:])
	if err != nil {
		return nil, err
	}
	epk, err := eosPub.Key()
	if err != nil {
		return nil, err
	}
	pub := ecies.ImportECDSAPublic(epk.ToECDSA())

	// derive the shared secret and hash it
	sharedKey, err := priv.GenerateShared(pub,32, 0)
	if err != nil {
		return nil, err
	}
	secretHash := sha512.New().Sum(sharedKey)

	// split our message into components
	signed := message[:len(message) - 64]
	encrypted := message[16:len(message) - 64]
	iv := message[:16]
	sig := message[len(message) - 64:]

	// check the signature
	verifier := hmac.New(sha256.New, secretHash[32:])
	if hex.EncodeToString(sig) != hex.EncodeToString(verifier.Sum(signed)) {
		return nil, errors.New("hmac signature is invalid")
	}

	// decrypt the message
	block, err := aes.NewCipher(secretHash[:32])
	if err != nil {
		return nil, err
	}
	cbc := cipher.NewCBCDecrypter(block, iv)
	plainText := make([]byte, len(encrypted))
	cbc.CryptBlocks(plainText, encrypted)
	padLen := int(plainText[len(plainText) - 1])
	if padLen >= len(plainText) {
		return nil, errors.New("invalid padding in message")
	}

	return plainText[:len(plainText) - padLen], nil
}