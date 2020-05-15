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
		pk, err := ecc.NewPublicKey("FIO"+priv.PublicKey().String()[3:])
		if err != nil {
			return nil, err
		}
		pks = append(pks, pk)
	}
	return pks, nil
}

