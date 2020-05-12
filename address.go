package fio

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eoscanada/eos-go"
	"github.com/fioprotocol/fio-go/eos-go/ecc"
	"io/ioutil"
	"math"
	"net/http"
	"regexp"
	"strings"
)

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
	if bad, err := regexp.MatchString(`(?:--|@@|@.*@|-@|@-|^-|-$)`, string(a)); bad || err != nil {
		return false
	}
	if match, err := regexp.MatchString(`^[a-zA-Z0-9-]+[@][a-zA-Z0-9-]+$`, string(a)); err != nil || !match {
		return false
	}
	return true
}

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

func NewRenewDomain(actor eos.AccountName, domain string) *Action {
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

// NewExpDomain is used by a test contract and not available on mainnet
//
// Deprecated: only used in development environments
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

// NewExpAddresses is used by a test contract and not available on mainnet
//
// Deprecated: only used in development environments
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
		TokenCode:  token,
		ChainCode:  chain,
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
	FioDomains   []FioName `json:"fio_domains,omitifempty"`
	FioAddresses []FioName `json:"fio_addresses,omitifempty"`
	Message      string    `json:"message,omitifempty"`
	More         uint32    `json:"more,omitifempty"`
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
	Limit        uint32 `json:"limit,omitempty"`
	Offset       uint32 `json:"offset,omitempty"`
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

func (api *API) getFioDomainsOrNames(endpoint string, pubKey string, offset uint32, limit uint32) (domains *FioNames, err error) {
	_, err = ActorFromPub(pubKey)
	if err != nil {
		return nil, err
	}
	req, err := json.Marshal(&getFioNamesRequest{
		FioPublicKey: pubKey,
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		return nil, err
	}
	resp, err := api.HttpClient.Post(api.BaseURL+"/v1/chain/"+endpoint, "application/json", bytes.NewReader(req))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("error %d: %s", resp.StatusCode, string(body)))
	}
	result := &FioNames{}
	err = json.Unmarshal(body, result)
	return result, err
}

// GetFioDomains queries for the domains owned by a Public Key. It offers paging which makes it preferable to GetFioNames
// which may not provide the full set of results because of (silent, without error) database query timeout issues.
// offset and limit must both be positive numbers. The returned uint32 specifies how many more results are available.
func (api *API) GetFioDomains(pubKey string, offset uint32, limit uint32) (domains *FioNames, err error) {
	return api.getFioDomainsOrNames("get_fio_domains", pubKey, offset, limit)
}

// GetFioAddresses queries for the FIO Addresses owned by a Public Key. It offers paging which makes it preferable to GetFioNames
// which may not provide the full set of results because of (silent, without error) database query timeout issues.
// offset and limit must both be positive numbers. The returned uint32 specifies how many more results are available.
func (api *API) GetFioAddresses(pubKey string, offset uint32, limit uint32) (domains *FioNames, err error) {
	return api.getFioDomainsOrNames("get_fio_addresses", pubKey, offset, limit)
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
	Account    *eos.AccountName `json:"account,omitempty"`
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

type AvailCheckReq struct {
	FioName string `json:"fio_name"`
}

type AvailCheckResp struct {
	IsRegistered uint8 `json:"is_registered"`
}

// AvailCheck responds with true if a domain or FIO address is available to be registered
func (api *API) AvailCheck(addressOrDomain string) (available bool, err error) {
	req := &AvailCheckReq{FioName: addressOrDomain}
	j, _ := json.Marshal(req)
	resp, err := api.HttpClient.Post(api.BaseURL+"/v1/chain/avail_check", "application/json", bytes.NewReader(j))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	isReg := &AvailCheckResp{}
	err = json.Unmarshal(body, isReg)
	if err != nil {
		return false, err
	}
	if isReg.IsRegistered == 0 {
		return true, nil
	}
	return false, nil
}

type AddressHistoryStatus uint8

const (
	AddressHistoryCurrent  AddressHistoryStatus = iota // the address is a valid address and is current
	AddressHistoryOutdated                             // the address has belonged to the pub key but is not current
	AddressHistoryInvalid                              // something nefarious is afoot
)

type VerifyAddressHistoryRequest struct {
	Pubkey     ecc.PublicKey
	FioAddress Address
	Token      string
	Chain      string
	PubAddress string

	// ChainId is optional, and can be empty. If empty the ChainIdMainnet const will be used (recommend leaving empty.)
	ChainId string

	// limit specifies how many actions will be searched for the addition of the address, use -1 to search all
	Limit int // FIXME: currently ignored.
}

// VerifyAddressHistory will search action traces for a pubkey and verify that:
// 1. the address was added by the correct actor
// 2. the transaction it was added in is valid and belongs to the correct chain.
//
// Note: this function requires access to a v1 history node and the get_actions API endpoint.
// There is a possibility that this will return a false negative (AddressHistoryInvalid) if the history
// node has truncated the actions for an account (see 'history-per-account' setting in config.ini).
func (api *API) VerifyAddressHistory(v VerifyAddressHistoryRequest) (AddressHistoryStatus, error) {
	if !api.HasHistory() {
		return AddressHistoryInvalid, errors.New("history plugin is not available")
	}
	if v.ChainId == "" {
		info, err := api.GetInfo()
		if err != nil {
			return AddressHistoryInvalid, err
		}
		if info.ChainID.String() != ChainIdMainnet {
			return AddressHistoryInvalid, errors.New("chain id is not for mainnet")
		}
		v.ChainId = ChainIdMainnet
	}

	status, txid, err := api.findAddress(v)
	if err != nil || status == AddressHistoryInvalid || txid == "" {
		return AddressHistoryInvalid, err
	}
	txidBytes, err := hex.DecodeString(txid)
	if err != nil {
		return AddressHistoryInvalid, err
	}

	// ensure tx is beyond lib
	tx, err := api.GetTransaction(txid)
	if err != nil {
		return AddressHistoryInvalid, err
	}
	info, err := api.GetInfo()
	if err != nil {
		return AddressHistoryInvalid, err
	}
	if info.LastIrreversibleBlockNum < tx.BlockNum {
		return AddressHistoryInvalid, errors.New("address might be valid, but transaction is not yet irreversible")
	}

	cid, err := hex.DecodeString(v.ChainId)
	if err != nil {
		return AddressHistoryInvalid, errors.New("could not decode chain id: "+err.Error())
	}

	block, err := api.GetBlockByNum(tx.BlockNum)
	if err != nil {
		return AddressHistoryInvalid, errors.New("get block containing tx: "+err.Error())
	}
	if block.Transactions == nil || len(block.Transactions) == 0 {
		return AddressHistoryInvalid, errors.New("block for tx was empty")
	}

	// TODO: validate the block signature
	// how can a block be authenticated given a few known items? And who's to say get_info isn't lying about the chainid?
	// - should we pull block #1 and validate the pubkey and date hashes to chainid? Probably.
	// - how can we be sure it's on the correct fork? do we need some sort of known-good block index?
	// - one possible solution is to verify all schedule changes, but not sure how to get this, only can get
	//   block number for the last change by looking at a block header state, can this be done without
	//   walking the entire chain? has to be a way, the p2p protocol supports header proofs via schedule validation.

	// validate tx signature,
	var foundInBlock bool
	for _, transaction := range block.Transactions {
		if !bytes.Equal(transaction.Transaction.ID, txidBytes) {
			continue
		}
		signedTx, err := transaction.Transaction.Packed.UnpackBare()
		if err != nil {
			fmt.Println(err)
			continue
		}
		// use chain id and tx signature to derived the signer's public key, if it matches our result is trustworthy
		signers, err := signedTx.SignedByKeys(cid)
		if err != nil {
			continue
		}
		for _, signer := range signers {
			// strip EOS and FIO to compare, SignedByKeys is using eos-go's ecc
			if signer.String()[3:] == v.Pubkey.String()[3:] {
				foundInBlock = true
			}
		}
	}
	if !foundInBlock {
		return AddressHistoryInvalid, errors.New("txid did not match block data")
	}

	return status, nil
}

// findAddress searches the trace history for the action, and provides whether it was added, and in what txid.
// an AddressHistoryInvalid or error here requires no further verification, otherwise the tx signatures should
// be checked. There are three ways an address mapping is created: addaddress, regaddress, and xferaddress,
// the latter two are mapped as an internal action in the contract, and the former-most is under user control.
// All three can be changed at a later time (via a new addaddress, or via xferaddress), likewise there are
// three possible outcomes: current, outdated, and invalid.
func (api *API) findAddress(v VerifyAddressHistoryRequest) (status AddressHistoryStatus, txid string, err error) {
	found := AddressHistoryInvalid
	account, err := ActorFromPub(v.Pubkey.String())
	if err != nil {
		return found, txid, err
	}
	highest, err := api.getMaxActions(account)
	if err != nil {
		return found, txid, err
	}
	if highest == 0 {
		return found, txid, nil
	}
	// search from the most recent, page at up to 100 records per request
	var alreadySeen bool
	a := eos.ActionResp{}

	// closure for searching addaddress action. This applies to address lookups for non-FIO chains, such as BTC, EOS, ETH
	searchAddAddress := func(addAddress *AddAddress) (hit bool, s AddressHistoryStatus, t string, e error) {
		for _, added := range addAddress.PublicAddresses {
			if addAddress.FioAddress == string(v.FioAddress) && added.TokenCode == v.Token && added.ChainCode == v.Chain {
				if added.PublicAddress == v.PubAddress {
					switch alreadySeen {
					case false:
						return true, AddressHistoryCurrent, a.Trace.TransactionID.String(), nil
					case true:
						return true, AddressHistoryOutdated, a.Trace.TransactionID.String(), nil
					}
				}
				alreadySeen = true
				hit = true
			}
		}
		return hit, AddressHistoryInvalid, "", nil
	}

	// closure for searching regaddress action. This *only* applies to validating FIO public key, the mapping
	// is created when any address is registered as part of the regaddress action.
	// two cases here, a new registration, or the address was transferred
	searchRegAddress := func(regAddress *RegAddress, transferAddress *TransferAddress) (hit bool, s AddressHistoryStatus, t string, e error) {
		if transferAddress != nil {
			switch transferAddress.NewOwnerFioPublicKey {
			case v.Pubkey.String():
				// if the first match is an incoming address transfer, we are done.
				return true, AddressHistoryCurrent, a.Trace.TransactionID.String(), nil
			default:
				alreadySeen = true
				return true, AddressHistoryOutdated, a.Trace.TransactionID.String(), nil
			}
		}
		if regAddress.FioAddress == string(v.FioAddress) && regAddress.OwnerFioPublicKey == v.Pubkey.String() {
			return true, AddressHistoryCurrent, a.Trace.TransactionID.String(), nil
		}
		return false, AddressHistoryInvalid, "", nil
	}

	for i := int64(highest); i > 0; i -= 100 {
		lower := i - 100
		if lower <= 0 {
			lower = 1
		}
		ar := &eos.ActionsResp{}
		ar, err = api.GetActions(eos.GetActionsRequest{
			AccountName: account,
			Pos:         lower,
			Offset:      i - lower,
		})
		if ar == nil || err != nil {
			break
		}
		var fioPub bool
		if strings.ToLower(v.Token) == "fio" && strings.ToLower(v.Chain) == "fio" {
			fioPub = true
		}
		if fioPub && v.PubAddress != v.Pubkey.String() {
			return AddressHistoryInvalid, "", errors.New("pub address must match FIO pub key when chain is FIO, and token is FIO")
		}

		// FIXME: use decrement to search most recent first!
		for _, a = range ar.Actions {
			if a.Trace.Action == nil {
				continue
			}
			switch fioPub {
			case true:
				if a.Trace.Action.Account == "fio.address" && ( a.Trace.Action.Name == "regaddress" || a.Trace.Action.Name == "xferaddress") {
					switch a.Trace.Action.Name {
					case "regaddress":
						regAddress := &RegAddress{}
						err = eos.UnmarshalBinary(a.Trace.Action.HexData, regAddress)
						if err != nil {
							break
						}
						var hit bool
						hit, status, txid, err = searchRegAddress(regAddress, nil)
						if !hit {
							break
						}
						if status == AddressHistoryCurrent {
							return
						}
					case "xferaddress":
						xferaddress := &TransferAddress{}
						err = eos.UnmarshalBinary(a.Trace.Action.HexData, xferaddress)
						if err != nil {
							break
						}
						var hit bool
						hit, status, txid, err = searchRegAddress(nil, xferaddress)
						if !hit {
							break
						}
						if status == AddressHistoryCurrent {
							return
						}
					}
				}
			default:
				if a.Trace.Action.Account == "fio.address" && a.Trace.Action.Name == "addpubaddress" {
					addAddress := &AddAddress{}
					err = eos.UnmarshalBinary(a.Trace.Action.HexData, addAddress)
					if err != nil {
						break
					}
					var hit bool
					hit, status, txid, err = searchAddAddress(addAddress)
					if !hit {
						break
					}
					if status == AddressHistoryCurrent {
						return
					}
				}
			}
		}
	}
	return found, txid, nil
}

// TODO: merge this into v1history branch, and out of address.go, this only exists for PoC

// accountActionSequence is a truncated action trace used only for finding the highest sequence number
type accountActionSequence struct {
	AccountActionSequence uint32 `json:"account_action_seq"`
}

type accountActions struct {
	Actions []accountActionSequence `json:"actions"`
}

// getMaxActions returns the highest account_action_sequence from the get_actions endpoint.
// This is needed because paging only works with positive offsets.
func (api *API) getMaxActions(account eos.AccountName) (highest uint32, err error) {
	resp, err := api.HttpClient.Post(
		api.BaseURL+"/v1/history/get_actions",
		"application/json",
		bytes.NewReader([]byte(fmt.Sprintf(`{"account_name":"%s","pos":-1}`, account))),
	)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	if len(body) == 0 {
		return 0, errors.New("received empty response")
	}

	aa := &accountActions{}
	err = json.Unmarshal(body, &aa)
	if err != nil {
		return 0, err
	}
	if aa.Actions == nil || len(aa.Actions) == 0 {
		return 0, nil
	}
	return aa.Actions[len(aa.Actions)-1].AccountActionSequence, nil
}

// HasHistory looks at available APIs and returns true if /v1/history/* exists.
func (api *API) HasHistory() bool {
	_, apis, err := api.GetSupportedApis()
	if err != nil {
		return false
	}
	for _, a := range apis {
		if strings.HasPrefix(a, `/v1/history/`) {
			return true
		}
	}
	return false
}
