package fio

import (
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/eoscanada/eos-go"
	eosecc "github.com/eoscanada/eos-go/ecc"
	"github.com/fioprotocol/fio-go/imports/ecc"
	hdwallet "github.com/fioprotocol/fio-go/imports/go-ethereum-hdwallet"
	"github.com/tyler-smith/go-bip32"
	mrand "math/rand"
	"strings"
	"time"
)

// Hd is an HD Wallet with BIP39 mnemonic phrase based on a BIP32 derivation path. Note: FIO uses m/44'/235'/0
type Hd struct {
	words  []string
	wallet *hdwallet.Wallet
}

// NewHdFromString verifies a mnemonic string and creates a Hd containing a HD Wallet
func NewHdFromString(mnemonic string) (*Hd, error) {
	mn := strings.Split(mnemonic, " ")
	switch len(mn) {
	case 12, 15, 18, 21, 24:
		for _, w := range mn {
			if w == "" {
				return nil, errors.New("malformed mnemonic, had empty word")
			}
		}
		break
	default:
		return nil, errors.New("mnemonic length should be 12, 15, 18, 21, or 24 words")
	}
	var result Hd
	var err error
	result.wallet, err = hdwallet.NewFromMnemonic(mnemonic)
	if err != nil {
		return nil, err
	}
	result.words = make([]string, len(mn))
	for i := range mn {
		result.words[i] = mn[i]
	}
	return &result, nil
}

// NewRandomHd builds a new Hd with a specific word count (12, 15, 18, 21, or 24,) longer is better
func NewRandomHd(words int) (*Hd, error) {
	var bits int
	switch words {
	case 24:
		bits = 256
	case 21:
		bits = 224
	case 18:
		bits = 192
	case 15:
		bits = 160
	case 12:
		bits = 128
	default:
		return nil, errors.New("word count must be 12, 15, 18, 21, or 24")
	}
	// confirmed as using crypto/rand not math:
	phrase, err := hdwallet.NewMnemonic(bits)
	if err != nil {
		return nil, err
	}
	return NewHdFromString(phrase)
}

// Xpriv is the bip32 root key as a string, this may not import for bip44 compatible wallets,
// that is a planned addition.
func (hd Hd) Xpriv() (string, error) {
	key, err := bip32.NewMasterKey(hd.wallet.Seed)
	if err != nil {
		return "", err
	}
	return key.String(), nil
}

// Xpub is the bip32 root public key as a string, some wallets will expect a bip44 xpub or the
// bip32 derivation xpub key, these are planned additions.
func (hd Hd) Xpub() (string, error) {
	key, err := bip32.NewMasterKey(hd.wallet.Seed)
	if err != nil {
		return "", err
	}
	return key.PublicKey().String(), nil
}

func (hd Hd) Len() int {
	return len(hd.words)
}

func (hd Hd) String() string {
	return strings.Join(hd.words[:], " ")
}

// Keys provides a keybag with the requested number of keys, use KeyAt for a single key
func (hd Hd) Keys(keys int) (*eos.KeyBag, error) {
	if keys < 1 {
		return nil, errors.New("cannot derive 0 keys")
	}
	keybag := &eos.KeyBag{}
	keybag.Keys = make([]*eosecc.PrivateKey, 0)
	for i := 0; i < keys; i++ {
		k, err := keyAt(hd.wallet, i)
		if err != nil {
			return nil, err
		}
		keybag.Keys = append(keybag.Keys, k)
	}
	return keybag, nil
}

// KeyAt creates a keybag holding a single key at m/44'/235'/0'/0/index
func (hd Hd) KeyAt(index int) (*eos.KeyBag, error) {
	keybag := &eos.KeyBag{}
	keybag.Keys = make([]*eosecc.PrivateKey, 1)
	var err error
	keybag.Keys[0], err = keyAt(hd.wallet, index)
	if err != nil {
		return nil, err
	}
	return keybag, nil
}

// PubKeys derives a number of public keys for the Hd
func (hd Hd) PubKeys(count int) ([]*ecc.PublicKey, error) {
	if count < 1 {
		return nil, errors.New("cannot derive 0 public keys")
	}
	privs, err := hd.Keys(count)
	if err != nil {
		return nil, err
	}
	pks := make([]*ecc.PublicKey, 0)
	for _, priv := range privs.Keys {
		pk, err := ecc.NewPublicKey("FIO" + priv.PublicKey().String()[3:])
		if err != nil {
			return nil, err
		}
		pks = append(pks, &pk)
	}
	return pks, nil
}

// PubKeyAt derives a public key at a specific location - m/44'/235'/0'/0/index
func (hd Hd) PubKeyAt(index int) (*ecc.PublicKey, error) {
	if index < 0 {
		return nil, errors.New("index must not be negative")
	}
	priv, err := hd.KeyAt(index)
	if err != nil {
		return nil, err
	}
	pk, err := ecc.NewPublicKey("FIO" + priv.Keys[0].PublicKey().String()[3:])
	if err != nil {
		return nil, err
	}
	return &pk, nil
}

func keyAt(wallet *hdwallet.Wallet, index int) (*eosecc.PrivateKey, error) {
	path, err := hdwallet.ParseDerivationPath(fmt.Sprintf("m/44'/235'/0'/0/%d", index))
	if err != nil {
		return nil, err
	}
	account, err := wallet.Derive(path, false)
	if err != nil {
		return nil, err
	}
	priv, err := wallet.PrivateKey(account)
	if err != nil {
		return nil, err
	}

	btcPriv := btcec.PrivateKey(*priv)
	wif, err := btcutil.NewWIF(&btcPriv, &chaincfg.MainNetParams, false)
	if err != nil {
		return nil, err
	}
	k, err := eosecc.NewPrivateKey(wif.String())
	if err != nil {
		return nil, err
	}
	return k, nil
}

// HdQuiz is used for prompting a user to confirm the mnemonic phrase by providing
// a description of which word to provide and a function to validate their answer
type HdQuiz struct {
	Description string
	Check       func(s string) bool // function confirming correct answer

	index int // for tests
	word  string
}

// Quiz generates a number of randomized quiz questions, if less than one is provided, it uses hd.Len()/3
func (hd Hd) Quiz(count int) (questions []HdQuiz, err error) {
	if count > hd.Len() {
		return nil, errors.New("invalid count requested, exceeds number of words")
	}
	if count < 1 {
		count = len(hd.words) / 3
	}
	for _, n := range hd.words {
		if n == "" {
			return nil, errors.New("invalid mnemonic, got an empty word")
		}
	}
	mrand.Seed(time.Now().UnixNano())
	chosen := make(map[int]bool)
	i := 0
	questions = make([]HdQuiz, count)
	for i < count {
		r := mrand.Intn(len(hd.words))
		if chosen[r] {
			continue
		}
		chosen[r] = true
		switch r {
		case 0:
			questions[i].Description = "first"
		case 1:
			questions[i].Description = "second"
		case 2:
			questions[i].Description = "third"
		case 3:
			questions[i].Description = "fourth"
		case 4:
			questions[i].Description = "fifth"
		case 5:
			questions[i].Description = "sixth"
		case 6:
			questions[i].Description = "seventh"
		case 7:
			questions[i].Description = "eighth"
		case 8:
			questions[i].Description = "ninth"
		case 9:
			questions[i].Description = "tenth"
		case 10:
			questions[i].Description = "eleventh"
		case 11:
			questions[i].Description = "twelfth"
		case 12:
			questions[i].Description = "thirteenth"
		case 13:
			questions[i].Description = "fourteenth"
		case 14:
			questions[i].Description = "fifteenth"
		case 15:
			questions[i].Description = "sixteenth"
		case 16:
			questions[i].Description = "seventeenth"
		case 17:
			questions[i].Description = "eighteenth"
		case 18:
			questions[i].Description = "nineteenth"
		case 19:
			questions[i].Description = "twentieth"
		case 20:
			questions[i].Description = "twenty-first"
		case 21:
			questions[i].Description = "twenty-second"
		case 22:
			questions[i].Description = "twenty-third"
		case 23:
			questions[i].Description = "twenty-fourth"
		}
		// closure ensures dereference of iterator
		func(i int, r int) {
			questions[i].word = hd.words[r]
			questions[i].index = r
			questions[i].Check = func(s string) bool {
				return strings.TrimSpace(s) == hd.words[r]
			}
		}(i, r)
		i += 1
	}
	return
}
