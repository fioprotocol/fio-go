package fio

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eoscanada/eos-go"
	"io/ioutil"
	"log"
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

// RegAddress simplifies the process of registering by making it a single step that waits for confirm
func (api *API) RegAddress(txOpts *TxOptions, actor *Account, ownerPub string, address string) (txId string, ok bool, err error) {
	action, ok := NewRegAddress(actor.Actor, Address(address), ownerPub)
	if !ok {
		return "", false, errors.New("invalid address")
	}
	tx := NewTransaction([]*Action{action}, txOpts)
	_, packedTx, err := api.SignTransaction(tx, txOpts.ChainID, CompressionNone)
	result, err := api.PushTransaction(packedTx)
	if err != nil {
		log.Println("push new address: " + err.Error())
		return "", false, err
	}
	_, err = api.WaitForConfirm(api.GetCurrentBlock()-2, result.TransactionID)
	if err != nil {
		log.Println("waiting for confirm: " + err.Error())
		return "", false, err
	}
	return result.TransactionID, true, nil
}

// AddAddress allows a public address of the specific blockchain type to be added to the FIO Address,
// so that it can be returned using /pub_address_lookup
type AddAddress struct {
	FioAddress      string          `json:"fio_address"`
	PublicAddresses []TokenPubAddr  `json:"public_addresses"`
	MaxFee          uint64          `json:"max_fee"`
	Actor           eos.AccountName `json:"actor"`
	Tpid            string          `json:"tpid"`
}

type TokenPubAddr struct {
	TokenCode     string `json:"token_code"`
	ChainCode     string `json:"chain_code"`
	PublicAddress string `json:"public_address"`
}

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

// RegDomain simplifies the process of registering by making it a single step that waits for confirm
//
// Deprecated: not an idiomatic implementation, other calls do not behave this way
func (api *API) RegDomain(txOpts *TxOptions, actor *Account, ownerPub string, domain string) (txId string, ok bool, err error) {
	action := NewRegDomain(actor.Actor, domain, ownerPub)
	tx := NewTransaction([]*Action{action}, txOpts)
	_, packedTx, err := api.SignTransaction(tx, txOpts.ChainID, CompressionNone)
	result, err := api.PushTransaction(packedTx)
	if err != nil {
		log.Println("push new domain: " + err.Error())
		return "", false, err
	}
	_, err = api.WaitForConfirm(api.GetCurrentBlock()-2, result.TransactionID)
	if err != nil {
		log.Println("waiting for domain: " + err.Error())
		return "", false, err
	}
	return result.TransactionID, true, nil
}

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

type BurnExpired struct{}

func NewBurnExpired(actor eos.AccountName) *Action {
	return NewAction(
		"fio.address", "burnexpired", actor,
		BurnExpired{},
	)
}

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

type FioNames struct {
	FioDomains   []FioName `json:"fio_domains"`
	FioAddresses []FioName `json:"fio_addresses"`
	Message      string    `json:"message,omitifempty"`
}

type FioName struct {
	FioDomain  string `json:"fio_domain,omitifempty"`
	FioAddress string `json:"fio_address,omitifempty"`
	Expiration string `json:"expiration"`
	IsPublic   int    `json:"is_public,omitifempty"`
}

type getFioNamesRequest struct {
	FioPublicKey string `json:"fio_public_key"`
}

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
	err = json.Unmarshal(resp.Rows, &found)
	if err != nil {
		return FioNames{}, false, err
	}
	if len(results) == 0 {
		return FioNames{}, false, errors.New("no matching account found in fio.address accountmap table")
	}
	return api.GetFioNames(results[0].Clientkey)
}
