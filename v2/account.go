package fio

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/blockpane/eos-go"
	"github.com/blockpane/eos-go/ecc"
	"github.com/mr-tron/base58"
	"strings"
)

// Account holds the information for an account, it differs from a regular EOS account in that the
// account name (Actor) is derived from the public key, and a FIO public key has a different prefix
type Account struct {
	KeyBag    *eos.KeyBag
	PubKey    string
	Actor     eos.AccountName
	Addresses []FioName
	Domains   []FioName
}

// Name wraps eos.Name for convenience and less imports for client
type Name eos.Name

func (n Name) ToEos() eos.Name {
	return eos.Name(n)
}

// NewAccountFromWif builds an Account given a private key string.
// Note: this is an ephemeral, in-memory, account which has no relation to keosd, and is not persistent.
func NewAccountFromWif(wif string) (*Account, error) {
	kb := eos.NewKeyBag()
	err := kb.ImportPrivateKey(context.Background(), wif)
	if err != nil {
		return nil, err
	}
	pub := kb.Keys[0].PublicKey().String()
	actor, err := ActorFromPub(pub)
	if err != nil {
		return nil, err
	}
	return &Account{
		KeyBag:    kb,
		PubKey:    pub,
		Actor:     actor,
		Addresses: make([]FioName, 0),
		Domains:   make([]FioName, 0),
	}, nil
}

// GetNames retrieves the FIO addresses and names owned by an account, and populates the Account struct
func (a *Account) GetNames(ctx context.Context, api *API) (addresses int, domains int, err error) {
	n, _, err := api.GetFioNames(ctx, a.PubKey)
	if err != nil {
		return 0, 0, nil
	}
	addresses = len(n.FioAddresses)
	domains = len(n.FioDomains)
	a.Addresses = n.FioAddresses
	a.Domains = n.FioDomains
	return
}

// NewRandomAccount creates a new account with a random key.
func NewRandomAccount() (*Account, error) {
	key, err := ecc.NewRandomPrivateKey()
	if err != nil {
		return nil, err
	}
	return NewAccountFromWif(key.String())
}

// ActorFromPub calculates the FIO Actor (EOS Account) from a public key
func ActorFromPub(pubKey string) (eos.AccountName, error) {
	// don't accept an EOS compat key
	if strings.HasPrefix(pubKey, "EOS") {
		return "", errors.New("expected FIO key, but got EOS prefix")
	}
	// ensure the key is valid base58, and the 160 checksum is correct before encoding
	p, err := ecc.NewPublicKey(pubKey)
	if err != nil {
		return "", err
	}
	pubKey = p.String() // ensure we end up with a compact key if we get PUB_K1_ prefix
	const actorKey = `.12345abcdefghijklmnopqrstuvwxyz`
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

// GetFioAccount gets information about an account, it should be used instead of GetAccount due to differences in
// public key formatting in eos vs fio packages.
func (api *API) GetFioAccount(ctx context.Context, actor string) (accResp *eos.AccountResp, err error) {
	err = api.call(ctx, "chain", "get_account", json.RawMessage(`{"account_name": "` + actor + `"}`), &accResp)
	return accResp, err
}

