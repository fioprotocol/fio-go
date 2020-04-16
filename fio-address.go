package fio

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eoscanada/eos-go"
	"io/ioutil"
	"math"
	"net/http"
)

const FioSymbol = "ᵮ"

// RegAddress Registers a FIO Address on the FIO blockchain
type RegAddress struct {
	FioAddress        string          `json:"fio_address"`
	OwnerFioPublicKey string          `json:"owner_fio_public_key"`
	MaxFee            uint64          `json:"max_fee"`
	Actor             eos.AccountName `json:"actor"`
	Tpid              string          `json:"tpid"`
}

func NewRegAddress(actor eos.AccountName, address Address, ownerPubKey string) (action *Action, ok bool) {
	if ok := address.Valid(); !ok {
		return nil, false
	}
	return NewAction(
		"fio.address", "regaddress", actor,
		RegAddress{
			FioAddress:        string(address),
			OwnerFioPublicKey: ownerPubKey,
			MaxFee:            Tokens(GetMaxFee(FeeRegisterFioAddress)),
			Actor:             actor,
			Tpid:              CurrentTpid(),
		},
	), true
}

// MustNewRegAddress panics on a bad address, but allows embedding because it only returns one value
func MustNewRegAddress(actor eos.AccountName, address Address, ownerPubKey string) (action *Action) {
	a, ok := NewRegAddress(actor, address, ownerPubKey)
	if !ok {
		panic("invalid fio address in call to MustNewRegAddress")
	}
	return a
}

// AddAddress allows a public address of the specific blockchain type to be added to the FIO Address,
// so that it can be returned using /pub_address_lookup
//
// When adding addresses, only 5 can be added in a single call, and an account is limited to 100 public
// addresses total.
type AddAddress struct {
	FioAddress      string          `json:"fio_address"`
	PublicAddresses []TokenPubAddr  `json:"public_addresses"`
	MaxFee          uint64          `json:"max_fee"`
	Actor           eos.AccountName `json:"actor"`
	Tpid            string          `json:"tpid"`
}

// TokenPubAddr holds *publicly* available token information for a FIO address, allowing anyone to lookup an address
type TokenPubAddr struct {
	TokenCode     string `json:"token_code"`
	ChainCode     string `json:"chain_code"`
	PublicAddress string `json:"public_address"`
}

// NewAddAddress adds a single public address
func NewAddAddress(actor eos.AccountName, fioAddress Address, token string, chain string, publicAddress string) (action *Action, ok bool) {
	if !fioAddress.Valid() {
		return nil, false
	}
	// ensure both chain and token are not empty
	if token != "" && chain == "" {
		token = chain
	} else if chain != "" && token == "" {
		chain = token
	} else if chain == "" && token == "" {
		return nil, false
	}
	return NewAction(
		"fio.address", "addaddress", actor,
		AddAddress{
			FioAddress:      string(fioAddress),
			PublicAddresses: []TokenPubAddr{{TokenCode: token, ChainCode: chain, PublicAddress: publicAddress}},
			MaxFee:          Tokens(GetMaxFee(FeeAddPubAddress)),
			Tpid:            CurrentTpid(),
			Actor:           actor,
		},
	), true
}

// NewAddAddresses adds multiple public addresses at a time
func NewAddAddresses(actor eos.AccountName, fioAddress Address, addrs []TokenPubAddr) (action *Action, ok bool) {
	if !fioAddress.Valid() {
		return nil, false
	}
	// fixup struct so both chain code and token code exist
	for _, a := range addrs {
		if a.TokenCode != "" && a.ChainCode == "" {
			a.TokenCode = a.ChainCode
		} else if a.ChainCode != "" && a.TokenCode == "" {
			a.ChainCode = a.TokenCode
		} else if a.ChainCode == "" && a.TokenCode == "" {
			return nil, false
		}
	}
	return NewAction(
		"fio.address", "addaddress", actor,
		AddAddress{
			FioAddress:      string(fioAddress),
			PublicAddresses: addrs,
			MaxFee:          Tokens(GetMaxFee(FeeAddPubAddress)),
			Tpid:            CurrentTpid(),
			Actor:           actor,
		},
	), true
}

// RegDomain registers a FIO Domain on the FIO blockchain
type RegDomain struct {
	FioDomain         string          `json:"fio_domain"`
	OwnerFioPublicKey string          `json:"owner_fio_public_key"`
	MaxFee            uint64          `json:"max_fee"`
	Actor             eos.AccountName `json:"actor"`
	Tpid              string          `json:"tpid"`
}

func NewRegDomain(actor eos.AccountName, domain string, ownerPubKey string) *Action {
	return NewAction(
		"fio.address", "regdomain", actor,
		RegDomain{
			FioDomain:         domain,
			OwnerFioPublicKey: ownerPubKey,
			MaxFee:            Tokens(GetMaxFee(FeeRegisterFioDomain)),
			Actor:             actor,
			Tpid:              CurrentTpid(),
		},
	)
}

// RenewDomain extends the expiration of a domain for a year
type RenewDomain struct {
	FioDomain string          `json:"fio_domain"`
	MaxFee    uint64          `json:"max_fee"`
	Tpid      string          `json:"tpid"`
	Actor     eos.AccountName `json:"actor"`
}

func NewRenewDomain(actor eos.AccountName, domain string, ownerPubKey string) *Action {
	return NewAction(
		"fio.address", "renewdomain", actor,
		RenewDomain{
			FioDomain: domain,
			MaxFee:    Tokens(GetMaxFee(FeeRenewFioDomain)),
			Actor:     actor,
			Tpid:      CurrentTpid(),
		},
	)
}

// TransferDom (future) transfers ownership of a domain
type TransferDom struct {
	FioDomain            string          `json:"fio_domain"`
	NewOwnerFioPublicKey string          `json:"new_owner_fio_public_key"`
	MaxFee               uint64          `json:"max_fee"`
	Tpid                 string          `json:"tpid"`
	Actor                eos.AccountName `json:"actor"`
}

func NewTransferDom(actor eos.AccountName, domain string, newOwnerPubKey string) *Action {
	return NewAction(
		"fio.address", "xferdomain", actor,
		TransferDom{
			FioDomain:            domain,
			NewOwnerFioPublicKey: newOwnerPubKey,
			MaxFee:               Tokens(GetMaxFee(FeeTransferDom)),
			Actor:                actor,
			Tpid:                 CurrentTpid(),
		},
	)
}

// RenewAddress extends the expiration of an address by a year, and refreshes the bundle
type RenewAddress struct {
	FioAddress string          `json:"fio_address"`
	MaxFee     uint64          `json:"max_fee"`
	Tpid       string          `json:"tpid"`
	Actor      eos.AccountName `json:"actor"`
}

func NewRenewAddress(actor eos.AccountName, address string) *Action {
	return NewAction(
		"fio.address", "renewaddress", actor,
		RenewAddress{
			FioAddress: address,
			MaxFee:     Tokens(GetMaxFee(FeeRenewFioAddress)),
			Tpid:       CurrentTpid(),
			Actor:      actor,
		},
	)
}

// TransferAddress (future) transfers ownership of a FIO address
type TransferAddress struct {
	FioAddress           string          `json:"fio_address"`
	NewOwnerFioPublicKey string          `json:"new_owner_fio_public_key"`
	MaxFee               uint64          `json:"max_fee"`
	Tpid                 string          `json:"tpid"`
	Actor                eos.AccountName `json:"actor"`
}

func NewTransferAddress(actor eos.AccountName, address Address, newOwnerPubKey string) *Action {
	return NewAction(
		"fio.address", "xferaddress", actor,
		TransferAddress{
			FioAddress:           string(address),
			NewOwnerFioPublicKey: newOwnerPubKey,
			MaxFee:               Tokens(GetMaxFee(FeeTransferAddress)),
			Actor:                actor,
			Tpid:                 CurrentTpid(),
		},
	)
}

// ExpDomain is used by a test contract and not available on mainnet
//
// Deprecated: only used in development environments
type ExpDomain struct {
	Actor  eos.AccountName `json:"actor"`
	Domain string          `json:"domain"`
}

func NewExpDomain(actor eos.AccountName, domain string) *Action {
	return NewAction(
		"fio.address", "expdomain", actor,
		ExpDomain{
			Actor:  actor,
			Domain: domain,
		},
	)
}

// ExpAddresses is used by a test contract and not available on mainnet
//
// Deprecated: only used in development environments
type ExpAddresses struct {
	Actor                eos.AccountName `json:"actor"`
	Domain               string          `json:"domain"`
	AddressPrefix        string          `json:"address_prefix"`
	NumberAddressesToAdd uint64          `json:"number_addresses_to_add"`
}

func NewExpAddresses(actor eos.AccountName, domain string, addressPrefix string, toAdd uint64) *Action {
	return NewAction(
		"fio.address", "expaddresses", actor,
		ExpAddresses{
			Actor:                actor,
			Domain:               domain,
			AddressPrefix:        addressPrefix,
			NumberAddressesToAdd: toAdd,
		},
	)
}

// BurnExpired is intended to be called by block producers to remove expired domains or addresses from RAM
type BurnExpired struct{}

func NewBurnExpired(actor eos.AccountName) *Action {
	return NewAction(
		"fio.address", "burnexpired", actor,
		BurnExpired{},
	)
}

// SetDomainPub changes the permissions for a domain, allowing (or not) anyone to register an address
type SetDomainPub struct {
	FioDomain string          `json:"fio_domain"`
	IsPublic  uint8           `json:"is_public"`
	MaxFee    uint64          `json:"max_fee"`
	Actor     eos.AccountName `json:"actor"`
	Tpid      string          `json:"tpid"`
}

func NewSetDomainPub(actor eos.AccountName, domain string, public bool) *Action {
	isPublic := 0
	if public {
		isPublic = 1
	}
	return NewAction(
		"fio.address", "setdomainpub", actor,
		SetDomainPub{
			FioDomain: domain,
			IsPublic:  uint8(isPublic),
			MaxFee:    Tokens(GetMaxFee(FeeSetDomainPub)),
			Actor:     actor,
			Tpid:      CurrentTpid(),
		},
	)
}

type PubAddress struct {
	PublicAddress string `json:"public_address"`
	Message       string `json:"message"`
}

type pubAddressRequest struct {
	FioAddress string `json:"fio_address"`
	TokenCode  string `json:"token_code"`
	ChainCode  string `json:"chain_code"`
}

// PubAddressLookup finds a public address for a user, given a currency key
//  pubAddress, ok, err := api.PubAddressLookup(fio.Address("alice:fio", "BTC")
func (api API) PubAddressLookup(fioAddress Address, chain string, token string) (address PubAddress, found bool, err error) {
	if token == "" {
		token = chain
	}
	if !fioAddress.Valid() {
		return PubAddress{}, false, errors.New("invalid fio address")
	}
	query := pubAddressRequest{
		FioAddress: string(fioAddress),
		TokenCode:  chain,
		ChainCode:  token,
	}
	j, _ := json.Marshal(query)
	req, err := http.NewRequest("POST", api.BaseURL+`/v1/chain/get_pub_address`, bytes.NewBuffer(j))
	if err != nil {
		return PubAddress{}, false, err
	}
	req.Header.Add("content-type", "application/json")
	res, err := api.HttpClient.Do(req)
	if err != nil {
		return PubAddress{}, false, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return PubAddress{}, false, err
	}
	err = json.Unmarshal(body, &address)
	if err != nil {
		return PubAddress{}, false, err
	}
	if address.PublicAddress != "" {
		found = true
	}
	return
}

// FioNames holds the response when getting fio names or addresses for an account
type FioNames struct {
	FioDomains   []FioName `json:"fio_domains"`
	FioAddresses []FioName `json:"fio_addresses"`
	Message      string    `json:"message,omitifempty"`
}

// FioName holds information for either an address or a domain
type FioName struct {
	FioDomain  string `json:"fio_domain,omitifempty"`
	FioAddress string `json:"fio_address,omitifempty"`
	Expiration string `json:"expiration"`
	IsPublic   int    `json:"is_public,omitifempty"`
}

type getFioNamesRequest struct {
	FioPublicKey string `json:"fio_public_key"`
}

// GetFioNames provides a list of domains and addresses for a public key
func (api API) GetFioNames(pubKey string) (names FioNames, found bool, err error) {
	query := getFioNamesRequest{
		FioPublicKey: pubKey,
	}
	j, _ := json.Marshal(query)
	req, err := http.NewRequest("POST", api.BaseURL+`/v1/chain/get_fio_names`, bytes.NewBuffer(j))
	if err != nil {
		return FioNames{}, false, err
	}
	req.Header.Add("content-type", "application/json")
	res, err := api.HttpClient.Do(req)
	if err != nil {
		return FioNames{}, false, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return FioNames{}, false, err
	}
	err = json.Unmarshal(body, &names)
	if err != nil {
		return FioNames{}, false, err
	}
	if len(names.FioAddresses) > 0 || len(names.FioDomains) > 0 {
		found = true
	}
	return
}

type accountMap struct {
	Clientkey string `json:"clientkey"`
}

// GetFioNamesForActor searches the accountmap table to get a public key, then searches for fio names or domains belonging
// to the associated public key
func (api *API) GetFioNamesForActor(actor string) (names FioNames, found bool, err error) {
	name, err := eos.StringToName(actor)
	if err != nil {
		return FioNames{}, false, err
	}
	resp, err := api.GetTableRows(eos.GetTableRowsRequest{
		Code:       "fio.address",
		Scope:      "fio.address",
		Table:      "accountmap",
		LowerBound: fmt.Sprintf("%d", name),
		UpperBound: fmt.Sprintf("%d", name),
		Limit:      math.MaxInt32,
		KeyType:    "i64",
		Index:      "1",
		JSON:       true,
	})
	if err != nil {
		return FioNames{}, false, err
	}
	results := make([]accountMap, 0)
	err = json.Unmarshal(resp.Rows, &results)
	if err != nil {
		return FioNames{}, false, err
	}
	if len(results) == 0 {
		return FioNames{}, false, errors.New("no matching account found in fio.address accountmap table")
	}
	return api.GetFioNames(results[0].Clientkey)
}

// DomainNameHash calculates the hash used as an index in the fio.address domains table from the domain name
func DomainNameHash(s string) string {
	sha := sha1.New()
	sha.Write([]byte(s))
	// last 16 bytes of sha1-sum, as big-endian
	return "0x" + hex.EncodeToString(flip(sha.Sum(nil)))[8:]
}

func flip(orig []byte) []byte {
	flipped := make([]byte, len(orig))
	for i := range orig {
		flipped[len(flipped)-i-1] = orig[i]
	}
	return flipped
}

// DomainResp holds the table query lookup result for a domain
type DomainResp struct {
	Name       string           `json:"name"`
	IsPublic   uint8            `json:"is_public"`
	Expiration int64            `json:"expiration"`
	Account    *eos.AccountName `json:"account"`
}

// GetDomainOwner finds the account that is the owner of a domain
func (api *API) GetDomainOwner(domain string) (actor *eos.AccountName, err error) {
	dnh := DomainNameHash(domain)
	resp, err := api.GetTableRows(eos.GetTableRowsRequest{
		Code:       "fio.address",
		Scope:      "fio.address",
		Table:      "domains",
		LowerBound: dnh,
		UpperBound: dnh,
		Limit:      1,
		KeyType:    "i128",
		Index:      "4",
		JSON:       true,
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Rows) < 2 {
		return nil, errors.New("not found")
	}
	d := make([]DomainResp, 0)
	err = json.Unmarshal(resp.Rows, &d)
	if err != nil {
		return nil, err
	}
	if len(d) == 0 {
		return nil, errors.New("not found")
	}
	return d[0].Account, nil
}
