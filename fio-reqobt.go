package fio

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/btcsuite/btcutil"
	"github.com/eoscanada/eos-go/ecc"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"io/ioutil"
	"net/http"
)

const (
	ObtInvalidType ObtType = iota
	ObtRequestType
	ObtResponseType
)

type ObtType uint8

func (o ObtType) String() string {
	switch o {
	case ObtRequestType:
		return "new_funds_content"
	case ObtResponseType:
		return "record_send_content"
	default:
		return ""
	}
}

// ObtRequestContent holds details for requesting funds
type ObtRequestContent struct {
	PayeePublicAddress string `json:"payee_public_address"`
	Amount             string `json:"amount"`
	ChainCode          string `json:"chain_code"`
	TokenCode          string `json:"token_code"`
	Memo               string `json:"memo"`
	Hash               string `json:"hash"`
	OfflineUrl         string `json:"offline_url"`
}

// ObtRequestContent holds details for requesting funds
type obtRequestContentOmit struct {
	PayeePublicAddress string `json:"payee_public_address"`
	Amount             string `json:"amount"`
	ChainCode          string `json:"chain_code"`
	TokenCode          string `json:"token_code"`
	Memo               string `json:"memo,omitempty"`
	Hash               string `json:"hash,omitempty"`
	OfflineUrl         string `json:"offline_url,omitempty"`
}

// Encrypt serializes and encrypts the 'content' field for OBT requests
func (req ObtRequestContent) Encrypt(from *Account, toPubKey string) (content string, err error) {
	reqOmit := obtRequestContentOmit{
		PayeePublicAddress: req.PayeePublicAddress,
		Amount:             req.Amount,
		ChainCode:          req.ChainCode,
		TokenCode:          req.TokenCode,
		Memo:               req.Memo,
		Hash:               req.Hash,
		OfflineUrl:         req.OfflineUrl,
	}
	j, err := json.Marshal(reqOmit)
	if err != nil {
		return "", err
	}
	abiReader := bytes.NewReader([]byte(obtAbiJsonOmit))
	abi, _ := eos.NewABI(abiReader)
	//bin, err := abi.DecodeTableRowTyped("record_send_content", j)
	bin, err := abi.EncodeAction("new_funds_content", j)
	if err != nil {
		return "", err
	}
	encrypted, err := EciesEncrypt(from, toPubKey, bin, nil)
	if err != nil {
		return "", err
	}
	return encrypted, nil
}

type ObtRecordContent struct {
	PayerPublicAddress string `json:"payer_public_address"`
	PayeePublicAddress string `json:"payee_public_address"`
	Amount             string `json:"amount"`
	ChainCode          string `json:"chain_code"`
	TokenCode          string `json:"token_code"`
	Status             string `json:"status"`
	ObtId              string `json:"obt_id"`
	Memo               string `json:"memo"`
	Hash               string `json:"hash"`
	OfflineUrl         string `json:"offline_url"`
}

type obtRecordContentOmit struct {
	PayerPublicAddress string `json:"payer_public_address"`
	PayeePublicAddress string `json:"payee_public_address"`
	Amount             string `json:"amount"`
	ChainCode          string `json:"chain_code"`
	TokenCode          string `json:"token_code"`
	Status             string `json:"status"`
	ObtId              string `json:"obt_id"`
	Memo               string `json:"memo,omitempty"`
	Hash               string `json:"hash,omitempty"`
	OfflineUrl         string `json:"offline_url,omitempty"`
}

// Encrypt serializes and encrypts the 'content' field for OBT requests
func (rec ObtRecordContent) Encrypt(from *Account, toPubKey string) (content string, err error) {
	recOmit := obtRecordContentOmit{
		rec.PayerPublicAddress,
		rec.PayeePublicAddress,
		rec.Amount,
		rec.ChainCode,
		rec.TokenCode,
		rec.Status,
		rec.ObtId,
		rec.Memo,
		rec.Hash,
		rec.OfflineUrl,
	}
	j, err := json.Marshal(recOmit)
	if err != nil {
		return "", err
	}
	abiReader := bytes.NewReader([]byte(obtAbiJsonOmit))
	abi, _ := eos.NewABI(abiReader)
	//bin, err := abi.DecodeTableRowTyped("record_send_content", j)
	bin, err := abi.EncodeAction("record_send_content", j)
	if err != nil {
		return "", err
	}
	encrypted, err := EciesEncrypt(from, toPubKey, bin, nil)
	if err != nil {
		return "", err
	}
	return encrypted, nil
}

type ObtContentResult struct {
	Type    ObtType
	Request *ObtRequestContent
	Record  *ObtRecordContent
}

func (c ObtContentResult) ToJson() ([]byte, error) {
	switch c.Type {
	case ObtRequestType:
		j, e := json.MarshalIndent(c.Request, "", "  ")
		if e != nil {
			return nil, e
		}
		return j, nil
	case ObtResponseType:
		j, e := json.MarshalIndent(c.Record, "", "  ")
		if e != nil {
			return nil, e
		}
		return j, nil
	}
	return nil, errors.New("unknown request type")
}

// DecryptContent provides a new populated ObtContentResult struct given an encrypted content payload
func DecryptContent(to *Account, fromPubKey string, encrypted string, obtType ObtType) (*ObtContentResult, error) {
	result := &ObtContentResult{
		Type: obtType,
	}

	bin, err := EciesDecrypt(to, fromPubKey, encrypted)
	if err != nil {
		return nil, err
	}
	switch obtType {
	case ObtRequestType:
		content, err := tryDecryptRequest(bin, obtType)
		if err != nil {
			return nil, err
		}
		result.Request = content
		return result, nil

	case ObtResponseType:
		content, err := tryDecryptRecord(bin, obtType)
		if err != nil {
			return nil, err
		}
		result.Record = content
		return result, nil
	}
	return nil, errors.New("unknown obtType: expecting fio.ObtResponseType or fio.ObtRequestType")
}

type RecordSend struct {
	FioRequestId    string `json:"fio_request_id"`
	PayerFioAddress string `json:"payer_fio_address"`
	PayeeFioAddress string `json:"payee_fio_address"`
	Content         string `json:"content"`
	MaxFee          uint64 `json:"max_fee"`
	Actor           string `json:"actor"`
	Tpid            string `json:"tpid"`
}

// NewRecordSend builds the action for providing the result of a off-chain transaction
func NewRecordSend(actor eos.AccountName, reqId string, payer string, payee string, content string) *Action {
	return NewAction(
		"fio.reqobt", "recordobt", actor,
		RecordSend{
			FioRequestId:    reqId,
			PayerFioAddress: payer,
			PayeeFioAddress: payee,
			Content:         content,
			MaxFee:          Tokens(GetMaxFee(FeeRecordObtData)),
			Actor:           string(actor),
			Tpid:            CurrentTpid(),
		},
	)
}

// FundsReq is a request sent from one user to another requesting funds
type FundsReq struct {
	PayerFioAddress string `json:"payer_fio_address"`
	PayeeFioAddress string `json:"payee_fio_address"`
	Content         string `json:"content"`
	MaxFee          uint64 `json:"max_fee"`
	Actor           string `json:"actor"`
	Tpid            string `json:"tpid"`
}

// FundsResp is a request sent from one user to another requesting funds, it includes the fio_request_id, so
// should be used when querying against the API endpoint
type FundsResp struct {
	PayerFioAddress string `json:"payer_fio_address"`
	PayeeFioAddress string `json:"payee_fio_address"`
	Content         string `json:"content"`
	MaxFee          uint64 `json:"max_fee"`
	Actor           string `json:"actor"`
	Tpid            string `json:"tpid"`
	FioRequestId    uint64 `json:"fio_request_id,omitempty"`
}

// NewFundsReq builds the action for providing the result of a off-chain transaction
func NewFundsReq(actor eos.AccountName, payerFio string, payeeFio string, content string) *Action {
	return NewAction(
		"fio.reqobt", "newfundsreq", actor,
		FundsReq{
			PayerFioAddress: payerFio,
			PayeeFioAddress: payeeFio,
			Content:         content,
			MaxFee:          Tokens(GetMaxFee(FeeNewFundsRequest)),
			Actor:           string(actor),
			Tpid:            CurrentTpid(),
		},
	)
}

// RejectFndReq is a response to a user, denying their request for funds.
type RejectFndReq struct {
	FioRequestId string `json:"fio_request_id"`
	MaxFee       uint64 `json:"max_fee"`
	Actor        string `json:"actor"`
	Tpid         string `json:"tpid"`
}

// NewRejectFndReq builds the action to reject a request
func NewRejectFndReq(actor eos.AccountName, requestId string) *Action {
	return NewAction(
		"fio.reqobt", "rejectfndreq", actor,
		RejectFndReq{
			FioRequestId: requestId,
			MaxFee:       Tokens(GetMaxFee(FeeRejectFundsRequest)),
			Actor:        string(actor),
			Tpid:         CurrentTpid(),
		},
	)
}

// EciesEncrypt implements the encryption format used in the content field of OBT requests.
//
// The plaintext is PKCS#7 padded before being encrypted -- returned output is base64
// Key derivation, and message format:
//
// A DH shared secret is created using ECIES (derives a key based on the curves of the public and private keys.)
// This secret is hashed *twice* using sha512, and the first 32 bytes of the hash is used to encrypt the message using
// AES-256 cbc, and the second half is used to create an outer sha256 hmac.
//
// The 16 byte IV is prepended to the output, resulting in the message format of:
//  IV + Ciphertext + HMAC
// See https://github.com/fioprotocol/fiojs/blob/master/docs/message_encryption.md for more information.
func EciesEncrypt(sender *Account, recipentPub string, plainText []byte, iv []byte) (content string, err error) {

	// Get the shared-secret
	_, secretHash, err := EciesSecret(sender, recipentPub)
	if err != nil {
		return "", err
	}
	hashAgain := sha512.New()
	_, err = hashAgain.Write(secretHash[:])
	if err != nil {
		return "", err
	}
	keys := hashAgain.Sum(nil)
	key := append(keys[:32])    // first half of sha512 hash of secret is used as key
	macKey := append(keys[32:]) // second half as hmac key

	// Generate IV
	var contentBuffer bytes.Buffer
	if len(iv) != 16 || bytes.Equal(iv, make([]byte, 16)) {
		iv = make([]byte, 16)
		_, err = rand.Read(iv)
		if err != nil {
			return "", err
		}
	}
	contentBuffer.Write(iv)

	// AES CBC for encryption,
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	cbc := cipher.NewCBCEncrypter(block, iv)

	//// create pkcs#7 padding
	plainText = append(plainText, func() []byte {
		padLen := block.BlockSize() - (len(plainText) % block.BlockSize())
		pad := make([]byte, padLen)
		for i := range pad {
			pad[i] = uint8(padLen)
		}
		return pad
	}()...)

	// encrypt the plaintext
	cipherText := make([]byte, len(plainText))
	cbc.CryptBlocks(cipherText, plainText)
	contentBuffer.Write(cipherText)

	// Sign the message using sha256 hmac, *second* half of sha512 hash used as key
	signer := hmac.New(sha256.New, macKey)
	_, err = signer.Write(contentBuffer.Bytes())
	if err != nil {
		return "", err
	}
	signature := signer.Sum(nil)
	contentBuffer.Write(signature)

	switch ObtOldFormat {
	case false:
		// base64 encode the message, and it's ready to be embedded in our FundsReq.Content or RecordSend.Content fields
		b64Buffer := bytes.NewBuffer([]byte{})
		//encoded := base64.NewEncoder(base64.URLEncoding, b64Buffer)
		encoded := base64.NewEncoder(base64.StdEncoding, b64Buffer)
		_, err = encoded.Write(contentBuffer.Bytes())
		_ = encoded.Close()
		return string(b64Buffer.Bytes()), nil
	default:
		return hex.EncodeToString(contentBuffer.Bytes()), nil
	}
}

// EciesDecrypt is the inverse of EciesEncrypt, using the recipient's private key and sender's public instead.
func EciesDecrypt(recipient *Account, senderPub string, message string) (decrypted []byte, err error) {
	const (
		sigLen = 32
	)

	var msg []byte
	switch ObtOldFormat {
	case false:
		// convert base64 string to []byte
		b64Reader := bytes.NewReader([]byte(message))
		//b64Decoder := base64.NewDecoder(base64.URLEncoding, b64Reader)
		b64Decoder := base64.NewDecoder(base64.StdEncoding, b64Reader)
		msg, err = ioutil.ReadAll(b64Decoder)
		if err != nil {
			return nil, err
		}
	default:
		// or the old style hex string
		msg, err = hex.DecodeString(message)
		if err != nil {
			return nil, err
		}
	}

	// Get the shared-secret
	_, secretHash, err := EciesSecret(recipient, senderPub)
	if err != nil {
		return nil, err
	}

	// Other SDK's hash it TWICE, so we will too ...
	hashTwice := sha512.New()
	_, err = hashTwice.Write(secretHash[:])
	if err != nil {
		return nil, err
	}
	secret := hashTwice.Sum(nil)

	// check the signature
	verifier := hmac.New(sha256.New, secret[32:])
	_, err = verifier.Write(msg[:len(msg)-sigLen])
	if err != nil {
		return nil, err
	}
	verified := verifier.Sum(nil)
	if hex.EncodeToString(msg[len(msg)-sigLen:]) != hex.EncodeToString(verified) {
		return nil,
			errors.New(
				fmt.Sprintf("hmac signature %s is invalid, expected %s",
					hex.EncodeToString(verified),
					hex.EncodeToString(msg[len(msg)-sigLen:]),
				),
			)
	}

	// decrypt the message
	block, err := aes.NewCipher(secret[:32])
	if err != nil {
		return nil, err
	}
	cbc := cipher.NewCBCDecrypter(block, msg[:block.BlockSize()])
	plainText := make([]byte, len(msg[block.BlockSize():len(msg)-sigLen]))
	cbc.CryptBlocks(plainText, msg[block.BlockSize():len(msg)-sigLen])
	if len(plainText) == 0 {
		return nil, errors.New("could not decrypt message")
	}

	padLen := int(plainText[len(plainText)-1])
	if padLen > block.BlockSize() || padLen >= len(plainText) {
		return nil, errors.New("invalid padding in message")
	}

	return plainText[:len(plainText)-padLen], nil
}

// depending on how the request was built it's possible to get a slightly different abi encoding,
// this will try three different ways of decoding the request ...
func tryDecryptRequest(bin []byte, obtType ObtType) (content *ObtRequestContent, err error) {
	content = &ObtRequestContent{}
	abiReader := bytes.NewReader([]byte(obtAbiJsonOmit))
	abi, _ := eos.NewABI(abiReader)
	decode, err := abi.DecodeTableRowTyped(obtType.String(), bin)
	if err != nil {
		abiReader = bytes.NewReader([]byte(obtAbiJson))
		abi, _ = eos.NewABI(abiReader)
		decode, err = abi.DecodeTableRowTyped(obtType.String(), bin)
		if err != nil {
			err := eos.UnmarshalBinary(bin, content)
			if err != nil {
				return nil, err
			}
		}
	}
	err = json.Unmarshal(decode, content)
	return
}

func tryDecryptRecord(bin []byte, obtType ObtType) (content *ObtRecordContent, err error) {
	content = &ObtRecordContent{}
	abiReader := bytes.NewReader([]byte(ObtAbiJson))
	abi, _ := eos.NewABI(abiReader)
	decode, err := abi.DecodeTableRowTyped(obtType.String(), bin)
	if err != nil {
		abiReader = bytes.NewReader([]byte(obtAbiJsonOmit))
		abi, _ = eos.NewABI(abiReader)
		decode, err = abi.DecodeTableRowTyped(obtType.String(), bin)
		if err != nil {
			err := eos.UnmarshalBinary(bin, content)
			if err != nil {
				return nil, err
			}
		}
	}
	err = json.Unmarshal(decode, content)
	return
}

// EciesSecret derives the ecies pre-shared key from a private and public key.
// The 'secret' returned is the actual secret, the 'hash' returned is what is actually used
// in the OBT implementation, allowing the secret to be stretched into two keys, one for
// encryption and one for message authentication.
func EciesSecret(private *Account, public string) (secret []byte, hash *[64]byte, err error) {
	// convert key to ecies private key type
	wif, err := btcutil.DecodeWIF(private.KeyBag.Keys[0].String())
	if err != nil {
		return nil, nil, err
	}
	priv := ecies.ImportECDSA(wif.PrivKey.ToECDSA())

	// convert public key string into an ecies public key struct
	eosPub, err := ecc.NewPublicKey(`EOS` + public[3:])
	if err != nil {
		return nil, nil, err
	}
	epk, err := eosPub.Key()
	if err != nil {
		return nil, nil, err
	}
	pub := ecies.ImportECDSAPublic(epk.ToECDSA())

	// derive the shared secret and hash it
	sharedKey, err := priv.GenerateShared(pub, 32, 0)
	if err != nil {
		return nil, nil, err
	}

	ss := sha512.Sum512(sharedKey)
	return sharedKey, &ss, nil
}

type getPendingFioNamesRequest struct {
	FioPublicKey string `json:"fio_public_key"`
	Limit        int    `json:"limit"`
	Offset       int    `json:"offset"`
}

type PendingFioRequestsResponse struct {
	Requests []FundsResp `json:"requests"`
	More     int         `json:"more"`
}

// GetPendingFioRequests looks for pending requests
func (api API) GetPendingFioRequests(pubKey string, limit int, offset int) (pendingRequests PendingFioRequestsResponse, hasPending bool, err error) {
	query := getPendingFioNamesRequest{
		FioPublicKey: pubKey,
		Limit:        limit,
		Offset:       offset,
	}
	j, err := json.Marshal(query)
	if err != nil {
		return PendingFioRequestsResponse{}, false, err
	}
	req, err := http.NewRequest("POST", api.BaseURL+`/v1/chain/get_pending_fio_requests`, bytes.NewBuffer(j))
	if err != nil {
		return PendingFioRequestsResponse{}, false, err
	}
	req.Header.Add("content-type", "application/json")
	res, err := api.HttpClient.Do(req)
	if err != nil {
		return PendingFioRequestsResponse{}, false, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return PendingFioRequestsResponse{}, false, err
	}
	err = json.Unmarshal(body, &pendingRequests)
	if err != nil {
		return PendingFioRequestsResponse{}, false, err
	}
	if len(pendingRequests.Requests) > 0 {
		hasPending = true
	}
	return
}

func swapEndian(orig []byte) (flipped []byte) {
	flipped = make([]byte, len(orig))
	for i := range orig {
		flipped[i] = orig[len(orig)-1-i]
	}
	return
}

// note, added non-existent actions for eos-go encoder ...
var ObtAbiJson = `{
    "version": "eosio::abi/1.0",
    "types": [],
    "actions": [{
         "name": "new_funds_content",
         "type": "new_funds_content",
         "ricardian_contract": ""
      },{
         "name": "record_send_content",
         "type": "record_send_content",
         "ricardian_contract": ""
      }
    ],
    "structs": [{
        "name": "new_funds_content",
        "base": "",
        "fields": [
            {"name": "payee_public_address", "type": "string"},
            {"name": "amount", "type": "string"},
            {"name": "token_code", "type": "string"},
            {"name": "memo", "type": "string"},
            {"name": "hash", "type": "string"},
            {"name": "offline_url", "type": "string"}
        ]
    }, {
        "name": "record_send_content",
        "base": "",
        "fields": [
            {"name": "payer_public_address", "type": "string"},
            {"name": "payee_public_address", "type": "string"},
            {"name": "amount", "type": "string"},
            {"name": "token_code", "type": "string"},
            {"name": "status", "type": "string"},
            {"name": "obt_id", "type": "string"},
            {"name": "memo", "type": "string"},
            {"name": "hash", "type": "string"},
            {"name": "offline_url", "type": "string"}
        ]
    }]
}
`

// note, added non-existent actions for eos-go encoder ...
var obtAbiJsonOmit = `{
    "version": "eosio::abi/1.0",
    "types": [],
    "actions": [{
         "name": "new_funds_content",
         "type": "new_funds_content",
         "ricardian_contract": ""
      },{
         "name": "record_send_content",
         "type": "record_send_content",
         "ricardian_contract": ""
      }
    ],
    "structs": [{
        "name": "new_funds_content",
        "base": "",
        "fields": [
            {"name": "payee_public_address", "type": "string"},
            {"name": "amount", "type": "string"},
            {"name": "chain_code", "type": "string"},
            {"name": "token_code", "type": "string"},
            {"name": "memo", "type": "string?"},
            {"name": "hash", "type": "string?"},
            {"name": "offline_url", "type": "string?"}
        ]
    }, {
        "name": "record_send_content",
        "base": "",
        "fields": [
            {"name": "payer_public_address", "type": "string"},
            {"name": "payee_public_address", "type": "string"},
            {"name": "amount", "type": "string"},
            {"name": "chain_code", "type": "string"},
            {"name": "token_code", "type": "string"},
            {"name": "status", "type": "string"},
            {"name": "obt_id", "type": "string"},
            {"name": "memo", "type": "string?"},
            {"name": "hash", "type": "string?"},
            {"name": "offline_url", "type": "string?"}
        ]
    }]
}`
