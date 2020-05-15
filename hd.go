package fio

import (
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/eoscanada/eos-go"
	eosecc "github.com/eoscanada/eos-go/ecc"
	"github.com/fioprotocol/fio-go/eos-go/ecc"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
	"math/rand"
	"strings"
	"time"
)

// HDNewKeys uses a BIP39 mnemonic phrase to generate a keybag with the specified number of keys based on a BIP32
// derivation path. Note: FIO uses m/44'/235'/0
func HDNewKeys(mnemonic string, keys int) (*eos.KeyBag, error) {
	if keys < 1 {
		return nil, errors.New("cannot derive 0 keys")
	}
	wallet, err := hdwallet.NewFromMnemonic(mnemonic)
	if err != nil {
		return nil, err
	}
	keybag := &eos.KeyBag{}
	keybag.Keys = make([]*eosecc.PrivateKey, 0)
	for i := 0; i < keys; i++ {
		path, err := hdwallet.ParseDerivationPath(fmt.Sprintf("m/44'/235'/0'/0/%d", i))
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
		keybag.Keys = append(keybag.Keys, k)
	}
	return keybag, nil
}

// HDGetPubKeys uses a BIP39 mnemonic phrase to generate a slice of public keys for the specified number of keys
// based on a BIP32 derivation path.
func HDGetPubKeys(mnemonic string, count int) ([]ecc.PublicKey, error) {
	if count < 1 {
		return nil, errors.New("cannot derive 0 public keys")
	}
	privs, err := HDNewKeys(mnemonic, count)
	if err != nil {
		return nil, err
	}
	pks := make([]ecc.PublicKey, 0)
	for _, priv := range privs.Keys {
		pk, err := ecc.NewPublicKey("FIO" + priv.PublicKey().String()[3:])
		if err != nil {
			return nil, err
		}
		pks = append(pks, pk)
	}
	return pks, nil
}

type Mnemonic []string

func MnemonicFromString(mnemonic string) (*Mnemonic, error) {
	mn := strings.Split(mnemonic, " ")
	switch len(mnemonic) {
	case 12, 15, 18, 21, 24:
		break
	default:
		return nil, errors.New("mnemonic length should be 12, 15, 18, 21, or 24 words")
	}
	_, err := hdwallet.NewFromMnemonic(mnemonic)
	if err != nil {
		return nil, err
	}
	var result Mnemonic
	for i := range mn {
		result[i] = mn[i]
	}
	return &result, nil
}

func (m Mnemonic) Len() int {
	return len(m)
}

func (m Mnemonic) String() string {
	return strings.Join(m[:], " ")
}

func (m Mnemonic) Keys(count int) (*eos.KeyBag, error) {
	return HDNewKeys(m.String(), count)
}

func (m Mnemonic) PubKeys(count int) ([]ecc.PublicKey, error) {
	return HDGetPubKeys(m.String(), count)
}

// MnemonicQuiz is used for prompting a user to confirm the mnemonic phrase by providing a partial description,
// the word, and a function to validate their answer
type MnemonicQuiz struct {
	Description string
	Word        string
	Correct     func(s string) bool
}

func (m Mnemonic) Quiz() (questions []MnemonicQuiz, err error) {
	for _, n := range m {
		if n == "" {
			return nil, errors.New("invalid mnemonic, got an empty word")
		}
	}
	rand.Seed(time.Now().UnixNano())
	chosen := make(map[int]bool)
	i := 0
	questions = make([]MnemonicQuiz, len(m)/3)
	for i < len(m)/3 {
		r := rand.Intn(len(m))
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
		questions[i].Word = m[i]
		func (i int, r int) {
			questions[i].Correct = func(s string) bool {
				return strings.TrimSpace(s) == m[r]
			}
		}(i, r)
		i += 1
	}
	return
}

