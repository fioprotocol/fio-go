package fio

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/eoscanada/eos-go"
	"io/ioutil"
	"log"
	"net/http"
)

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
	PublicAddress string `json:"public_address"`
}

func NewAddAddress(actor eos.AccountName, fioAddress Address, token string, publicAddress string) (action *Action, ok bool) {
	if ok := fioAddress.Valid(); !ok {
		return nil, false
	}
	return NewAction(
		"fio.address", "addaddress", actor,
		AddAddress{
			FioAddress:      string(fioAddress),
			PublicAddresses: []TokenPubAddr{{token, publicAddress}},
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
}

// PubAddressLookup finds a public address for a user, given a currency key
//  pubAddress, ok, err := api.PubAddressLookup(fio.Address("alice:fio", "BTC")
func (api API) PubAddressLookup(fioAddress Address, chain string) (address PubAddress, found bool, err error) {
	if !fioAddress.Valid() {
		return PubAddress{}, false, errors.New("invalid fio address")
	}
	query := pubAddressRequest{
		FioAddress: string(fioAddress),
		TokenCode:  chain,
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
