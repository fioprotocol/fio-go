package fio

import (
	"errors"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	"github.com/mr-tron/base58"
	"regexp"
	"strings"
)

// Account holds the information for an account, it differs from a regular EOS account in that the
// account name (Actor) is derived from the public key, and a FIO public key has a different prefix
type Account struct {
	KeyBag    *eos.KeyBag
	PubKey    string
	Actor     eos.AccountName
	Addresses []string
}

// NewAccountFromWif builds an Account given a private key string
func NewAccountFromWif(wif string) (*Account, error) {
	kb := eos.NewKeyBag()
	err := kb.ImportPrivateKey(wif)
	if err != nil {
		return nil, err
	}
	pub := PubFromEos(kb.Keys[0].PublicKey().String())
	actor, err := ActorFromPub(pub)
	if err != nil {
		return nil, err
	}
	return &Account{
		KeyBag:    kb,
		PubKey:    pub,
		Actor:     actor,
		Addresses: make([]string, 0),
	}, nil
}

func NewRandomAccount() (*Account, error) {
	key, err := ecc.NewRandomPrivateKey()
	if err != nil {
		return nil, err
	}
	return NewAccountFromWif(key.String())
}

const actorKey = `.12345abcdefghijklmnopqrstuvwxyz`

// ActorFromPub calculates the FIO Actor (EOS Account) from a public key
func ActorFromPub(pubKey string) (eos.AccountName, error) {
	if len(pubKey) != 53 {
		return "", errors.New("public key should be 53 chars")
	}
	decoded, err := base58.Decode(pubKey[3:])
	if err != nil {
		return "", err
	}
	var result uint64
	i := 1
	for found := 0; found <= 12; i++ {
		if i > 32 {
			return "", errors.New("key has more than 20 bytes with trailing zeros")
		}
		var n uint64
		if found == 12 {
			n = uint64(decoded[i]) & uint64(0x0f)
		} else {
			n = uint64(decoded[i]) & uint64(0x1f) << uint64(5*(12-found)-1)
		}
		if n == 0 {
			continue
		}
		result = result | n
		found = found + 1
	}
	actor := make([]byte, 13)
	actor[12] = actorKey[result&uint64(0x0f)]
	result = result >> 4
	for i := 1; i <= 12; i++ {
		actor[12-i] = actorKey[result&uint64(0x1f)]
		result = result >> 5
	}
	return eos.AccountName(string(actor[:12])), nil
}

type Address string

// Valid checks for the correct address format
/*
  String
  Min: 3
  Max: 64
  Characters allowed: ASCII a-z0-9 - (dash) : (colon)
  Characters required:
     only one : (colon) and at least one a-z0-9 on either side of colon.
     a-z0-9 is required on either side of any dash
  Case-insensitive
*/
func (a Address) Valid() (ok bool) {
	if len(string(a)) < 3 || len(string(a)) > 64 {
		return false
	}
	if bad, err := regexp.MatchString(`(?:--|::|:.*:|-:|:-|^-|-$)`, string(a)); bad || err != nil {
		return false
	}
	if match, err := regexp.MatchString(`[a-zA-Z0-9-]+:[a-zA-Z0-9-]`, string(a)); err != nil || !match {
		return false
	}
	return true
}

// PubFromEos is a convenience function that returns the FIO pub address from an EOS pub address
func PubFromEos(eosPub string) (fioPub string) {
	return strings.Replace(eosPub, "EOS", "FIO", 1)
}
