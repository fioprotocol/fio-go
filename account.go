package fio

import (
	"bytes"
	"encoding/json"
	"errors"
	feos "github.com/fioprotocol/fio-go/imports/eos-fio"
	"github.com/fioprotocol/fio-go/imports/eos-fio/fecc"
	"github.com/mr-tron/base58"
	"io/ioutil"
)

// Account holds the information for an account, it differs from a regular EOS account in that the
// account name (Actor) is derived from the public key, and a FIO public key has a different prefix
type Account struct {
	KeyBag    *feos.KeyBag
	PubKey    string
	Actor     feos.AccountName
	Addresses []FioName
	Domains   []FioName
}

// Name wraps eos.Name for convenience and less imports for client
type Name feos.Name

func (n Name) ToEos() feos.Name {
	return feos.Name(n)
}

// NewAccountFromWif builds an Account given a private key string.
// Note: this is an ephemeral, in-memory, account which has no relation to keosd, and is not persistent.
func NewAccountFromWif(wif string) (*Account, error) {
	kb := feos.NewKeyBag()
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
	key, err := fecc.NewRandomPrivateKey()
	if err != nil {
		return nil, err
	}
	return NewAccountFromWif(key.String())
}

// ActorFromPub calculates the FIO Actor (EOS Account) from a public key
func ActorFromPub(pubKey string) (feos.AccountName, error) {
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
	return feos.AccountName(string(actor[:12])), nil
}

/*
	the following override the eos-go ecc library to handle the FIO prefix, this avoids errors during
	deserialization
*/

// AccountResp duplicates the eos.AccountResp accounting for differences in public key format
type AccountResp struct {
	AccountName            feos.AccountName          `json:"account_name"`
	Privileged             bool                      `json:"privileged"`
	LastCodeUpdate         feos.JSONTime             `json:"last_code_update"`
	Created                feos.JSONTime             `json:"created"`
	CoreLiquidBalance      feos.Asset                `json:"core_liquid_balance"`
	RAMQuota               feos.Int64                `json:"ram_quota"`
	RAMUsage               feos.Int64                `json:"ram_usage"`
	NetWeight              feos.Int64                `json:"net_weight"`
	CPUWeight              feos.Int64                `json:"cpu_weight"`
	NetLimit               feos.AccountResourceLimit `json:"net_limit"`
	CPULimit               feos.AccountResourceLimit `json:"cpu_limit"`
	Permissions            []Permission              `json:"permissions"`
	TotalResources         feos.TotalResources       `json:"total_resources"`
	SelfDelegatedBandwidth feos.DelegatedBandwidth   `json:"self_delegated_bandwidth"`
	RefundRequest          *feos.RefundRequest       `json:"refund_request"`
	VoterInfo              feos.VoterInfo            `json:"voter_info"`
}

// Permission duplicates the eos.Permission accounting for differences in public key format
type Permission struct {
	PermName     string    `json:"perm_name"`
	Parent       string    `json:"parent"`
	RequiredAuth Authority `json:"required_auth"`
}

// Authority duplicates the eos.Authority accounting for differences in public key format
type Authority struct {
	Threshold uint32                       `json:"threshold"`
	Keys      []KeyWeight                  `json:"keys,omitempty"`
	Accounts  []feos.PermissionLevelWeight `json:"accounts,omitempty"`
	Waits     []feos.WaitWeight            `json:"waits,omitempty"`
}

// KeyWeight duplicates the eos.KeyWeight accounting for differences in public key format
type KeyWeight struct {
	PublicKey fecc.PublicKey `json:"key"`
	Weight    uint16         `json:"weight"` // weight_type
}

// GetFioAccount gets information about an account, it should be used instead of GetAccount due to differences in
// public key formatting in eos vs fio packages.
func (api *API) GetFioAccount(actor string) (*AccountResp, error) {
	q := bytes.NewReader([]byte(`{"account_name": "` + actor + `"}`))
	resp, err := api.HttpClient.Post(api.BaseURL+"/v1/chain/get_account", "application/json", q)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	accResp := &AccountResp{}
	err = json.Unmarshal(body, accResp)
	if err != nil && err.Error() == `public key should start with "FIO"` {
		err = nil
	}
	return accResp, err
}

// pubFromEos is a convenience function that returns the FIO pub address from an EOS pub address
func pubFromEos(eosPub string) (fioPub string) {
	return "FIO" + eosPub[3:]
}
