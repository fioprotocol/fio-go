package fio

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	"github.com/mr-tron/base58"
	"io/ioutil"
	"regexp"
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
	err := kb.ImportPrivateKey(wif)
	if err != nil {
		return nil, err
	}
	pub := pubFromEos(kb.Keys[0].PublicKey().String())
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
func (a *Account) GetNames(api *API) (addresses int, domains int, err error) {
	n, _, err := api.GetFioNames(a.PubKey)
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

// Address is a FIO address, which should be formatted as 'name@domain'
type Address string

// Valid checks for the correct fio.Address formatting
//  Rules:
//    Min: 3
//    Max: 64
//    Characters allowed: ASCII a-z0-9 - (dash) @ (ampersat)
//    Characters required:
//       only one @ and at least one a-z0-9 on either side of @.
//       a-z0-9 is required on either side of any dash
//    Case-insensitive
func (a Address) Valid() (ok bool) {
	if len(string(a)) < 3 || len(string(a)) > 64 {
		return false
	}
	if bad, err := regexp.MatchString(`(?:--|::|:.*:|-:|:-|^-|-$)`, string(a)); bad || err != nil {
		return false
	}
	if bad, err := regexp.MatchString(`(?:--|@@|@.*@|-@|@-|^-|-$)`, string(a)); bad || err != nil {
		return false
	}
	if match, err := regexp.MatchString(`[a-zA-Z0-9-]+[:@][a-zA-Z0-9-]`, string(a)); err != nil || !match {
		return false
	}
	return true
}

func (api *API) GetFioAccount(actor string) (*eos.AccountResp, error) {
	q := bytes.NewReader([]byte(`{"account_name": "`+actor+`"}`))
	resp, err := api.HttpClient.Post(api.BaseURL+"/v1/chain/get_account", "application/json", q)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	accResp := &eos.AccountResp{}
	err = json.Unmarshal(body, accResp)
	if err != nil && err.Error() == `public key should start with ["PUB_K1_" | "PUB_R1_"] (or the old "EOS")` {
		err = nil
	}
	return accResp, err
}

// pubFromEos is a convenience function that returns the FIO pub address from an EOS pub address
func pubFromEos(eosPub string) (fioPub string) {
	return strings.Replace(eosPub, "EOS", "FIO", 1)
}
