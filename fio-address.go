package fio

import "github.com/eoscanada/eos-go"

// RegAddress Registers a FIO Address on the FIO blockchain
type RegAddress struct {
	FioAddress        string          `json:"fio_address"`
	OwnerFioPublicKey string          `json:"owner_fio_public_key"`
	MaxFee            uint64          `json:"max_fee"`
	Actor             eos.AccountName `json:"actor"`
	Tpid              string          `json:"tpid"`
}

func NewRegAddress(actor eos.AccountName, address Address, ownerPubKey string) (action *eos.Action, ok bool) {
	if ok := address.Valid(); !ok {
		return nil, false
	}
	return newAction(
		"fio.address", "regaddress", actor,
		RegAddress{
			FioAddress:        string(address),
			OwnerFioPublicKey: ownerPubKey,
			MaxFee:            Tokens(GetMaxFee("register_fio_address")),
			Actor:             actor,
			Tpid:              globalTpid,
		},
	), true
}

// AddAddress allows a public address of the specific blockchain type to be added to the FIO Address,
// so that it can be returned using /pub_address_lookup
type AddAddress struct {
	FioAddress    string          `json:"fio_address"`
	TokenCode     string          `json:"token_code"`
	PublicAddress string          `json:"public_address"`
	MaxFee        uint64          `json:"max_fee"`
	Actor         eos.AccountName `json:"actor"`
	Tpid          string          `json:"tpid"`
}

func NewAddAddress(actor eos.AccountName, fioAddress Address, token string, publicAddress string) (action *eos.Action, ok bool) {
	if ok := fioAddress.Valid(); !ok {
		return nil, false
	}
	return newAction(
		"fio.address", "addaddress", actor,
		AddAddress{
			FioAddress:    string(fioAddress),
			TokenCode:     token,
			PublicAddress: publicAddress,
			MaxFee:        Tokens(GetMaxFee("add_pub_address")),
			Tpid:          globalTpid,
			Actor:         actor,
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

func NewRegDomain(actor eos.AccountName, domain string, ownerPubKey string) *eos.Action {
	return newAction(
		"fio.address", "regdomain", actor,
		RegDomain{
			FioDomain:         domain,
			OwnerFioPublicKey: ownerPubKey,
			MaxFee:            Tokens(GetMaxFee("register_fio_domain")),
			Actor:             actor,
			Tpid:              globalTpid,
		},
	)
}

type RenewDomain struct {
	FioDomain string          `json:"fio_domain"`
	MaxFee    uint64          `json:"max_fee"`
	Tpid      string          `json:"tpid"`
	Actor     eos.AccountName `json:"actor"`
}

func NewRenewDomain(actor eos.AccountName, domain string, ownerPubKey string) *eos.Action {
	return newAction(
		"fio.address", "renewdomain", actor,
		RenewDomain{
			FioDomain: domain,
			MaxFee:    Tokens(GetMaxFee("renew_fio_domain")),
			Actor:     actor,
			Tpid:      globalTpid,
		},
	)
}

type RenewAddress struct {
	FioAddress string          `json:"fio_address"`
	MaxFee     uint64          `json:"max_fee"`
	Tpid       string          `json:"tpid"`
	Actor      eos.AccountName `json:"actor"`
}

func NewRenewAddress(actor eos.AccountName, address string) *eos.Action {
	return newAction(
		"fio.address", "renewaddress", actor,
		RenewAddress{
			FioAddress: address,
			MaxFee:     Tokens(GetMaxFee("renew_fio_address")),
			Tpid:       globalTpid,
			Actor:      actor,
		},
	)
}

type ExpDomain struct {
	Actor  eos.AccountName `json:"actor"`
	Domain string          `json:"domain"`
}

func NewExpDomain(actor eos.AccountName, domain string) *eos.Action {
	return newAction(
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

func NewExpAddresses(actor eos.AccountName, domain string, addressPrefix string, toAdd uint64) *eos.Action {
	return newAction(
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

func NewBurnExpired(actor eos.AccountName) *eos.Action {
	return newAction(
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

func NewSetDomainPub(actor eos.AccountName, domain string, public bool) *eos.Action {
	isPublic := 0
	if public {
		isPublic = 1
	}
	return newAction(
		"fio.address", "setdomainpub", actor,
		SetDomainPub{
			FioDomain: domain,
			IsPublic:  uint8(isPublic),
			MaxFee:    Tokens(GetMaxFee("setdomainpub")),
			Actor:     actor,
			Tpid:      globalTpid,
		},
	)
}
