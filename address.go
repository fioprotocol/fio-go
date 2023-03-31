package fio

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fioprotocol/fio-go/eos"
	"io/ioutil"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"crypto/sha1" // #nosec
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

// BurnAddress will destroy an address owned by an account
type BurnAddress struct {
	FioAddress string          `json:"fio_address"`
	Actor      eos.AccountName `json:"actor"`
	Tpid       string          `json:"tpid"`
	MaxFee     uint64          `json:"max_fee"`
}

func NewBurnAddress(actor eos.AccountName, address Address) (action *Action, ok bool) {
	if ok := address.Valid(); !ok {
		return nil, false
	}
	return NewAction("fio.address", "burnaddress", actor,
		BurnAddress{
			FioAddress: string(address),
			Actor:      actor,
			Tpid:       CurrentTpid(),
			MaxFee:     Tokens(GetMaxFee(FeeBurnAddress)),
		},
	), true
}

func MustNewBurnAddress(actor eos.AccountName, address Address) (action *Action) {
	a, ok := NewBurnAddress(actor, address)
	if !ok {
		panic("invalid fio address in call to MustBurnNewAddress")
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
	Actor                eos.AccountName `json:"actor"`
	Tpid                 string          `json:"tpid"`
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

// BurnExpired is intended to be called by block producers to remove expired domains or addresses from RAM
//
// Deprecated: as of the 2.5.x contracts release this will not work, use BurnExpiredRange instead
type BurnExpired struct{}

// NewBurnExpired will return a burnexpired action. It has been updated to return a BurnExpiredRange action
// as of the 3.1.x release. It didn't work before, and this prevents a breaking change in existing clients but will not work.
//
// Deprecated: this is essentially a noop, use GetExpiredOffset to find the offset, and provide it to NewBurnExpiredRange
func NewBurnExpired(actor eos.AccountName) *Action {
	return NewBurnExpiredRange(0, 15, actor)
}

// NewBurnExpiredRange will return a burnexpired action. The offset should be the first ID in the domains
// table that is expired. The GetExpiredOffset helper makes this easier.
//	offset, err := api.GetExpiredOffset(false)
//	if err != nil {
//	    ...
//	}
//	_, err = api.SignPushActions(fio.NewBurnExpiredRange(offset, 15, acc.Actor))
//	if err != nil {
//	    ...
//	}
func NewBurnExpiredRange(offset int64, limit int32, actor eos.AccountName) *Action {
	return NewAction(
		"fio.address", "burnexpired", actor,
		BurnExpiredRange{
			Offset: offset,
			Limit:  limit,
		},
	)
}

// BurnExpiredRange is intended to be called by block producers to remove expired domains or addresses from RAM
type BurnExpiredRange struct {
	Offset int64 `json:"offset"`
	Limit  int32 `json:"limit"`
}

type expiredIdOnly struct {
	Id int64 `json:"id"`
}

// GetExpiredOffset finds the first FIO domain in the fio.address::domains table, and returns the index for that domain.
func (api *API) GetExpiredOffset(descending bool) (int64, error) {
	gtro, err := api.GetTableRowsOrder(GetTableRowsOrderRequest{
		Code:       "fio.address",
		Scope:      "fio.address",
		Table:      "domains",
		LowerBound: "0",
		UpperBound: strconv.Itoa(int(time.Now().Add(-90*24*time.Hour).UTC().Unix())),
		Limit:      1,
		KeyType:    "i64",
		Index:      "3",
		JSON:       true,
		Reverse:    descending,
	})
	if err != nil {
		return 0, err
	}
	ids := make([]expiredIdOnly, 0)
	err = json.Unmarshal(gtro.Rows, &ids)
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 || ids[0].Id == 0 {
		return 0, errors.New("no results for GetExpiredOffset")
	}
	return ids[0].Id, nil
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

// GetPublic is an alias to PubAddressLookup to correct the confusing name for the lookup.
func (api *API) GetPublic(fioAddress Address, chain string, token string) (address PubAddress, found bool, err error) {
	return api.PubAddressLookup(fioAddress, chain, token)
}

type getAllPublicResp struct {
	Addresses []TokenPubAddr `json:"addresses"`
}

// GetAllPublic fetches all public addresses for an address.
func (api *API) GetAllPublic(fioAddress Address) ([]TokenPubAddr, error) {
	gtr, err := api.GetTableRows(eos.GetTableRowsRequest{
		Code:       "fio.address",
		Scope:      "fio.address",
		Table:      "fionames",
		LowerBound: I128Hash(string(fioAddress)),
		UpperBound: I128Hash(string(fioAddress)),
		KeyType:    "i128",
		Index:      "5",
		Limit:      1,
		JSON:       true,
	})
	if err != nil {
		return nil, err
	}
	result := make([]getAllPublicResp, 0)
	err = json.Unmarshal(gtr.Rows, &result)
	if len(result) == 0 {
		return nil, errors.New("empty result")
	}
	return result[0].Addresses, nil
}

// PubAddressLookup finds a public address for a user, given a currency key
//  pubAddress, ok, err := api.PubAddressLookup(fio.Address("alice:fio", "BTC")
func (api *API) PubAddressLookup(fioAddress Address, chain string, token string) (address PubAddress, found bool, err error) {
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
func (api *API) GetFioNames(pubKey string) (names FioNames, found bool, err error) {
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
func (api *API) GetFioAddresses(pubKey string, offset uint32, limit uint32) (addresses *FioNames, err error) {
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

// I128Hash hashes a string to an i128 database value, often used as an index for a string in a table.
// It is the most-significant 16 bytes in big-endian of a sha1 hash of the provided string, returned as a hex-string
func I128Hash(s string) string {
	sha := sha1.New() // #nosec
	_, err := sha.Write([]byte(s))
	if err != nil {
		return ""
	}
	// last 16 bytes of sha1-sum, as big-endian
	return "0x" + hex.EncodeToString(flip(sha.Sum(nil)))[8:]
}

// DomainNameHash calculates the hash used as index 4 in the fio.address domains table from the domain name.
// This is an alias to I128Hash. Example for domain `fio`:
//    {
//      "code": "fio.address",
//      "scope": "fio.address",
//      "table": "domains",
//      "lower_bound": "0x8d9d3bd8a6fb22345ce8fa3c416a28e5",
//      "upper_bound": "0x8d9d3bd8a6fb22345ce8fa3c416a28e5",
//      "key_type": "i128",
//      "index_position": "4",
//      "json": true
//    }
func DomainNameHash(s string) string {
	return I128Hash(s)
}

// AddressHash calculates the hash used as index 5 in the fio.address fionames table from the domain name.
// This is an alias to I128Hash, example of query for `test@fiotestnet`:
//    {
//      "code": "fio.address",
//      "scope": "fio.address",
//      "table": "fionames",
//      "lower_bound": "0xeb0816aeb936141ebec9a4a76c64df58",
//      "upper_bound": "0xeb0816aeb936141ebec9a4a76c64df58",
//      "key_type": "i128",
//      "index_position": "5",
//      "json": true
//    }
func AddressHash(s string) string {
	return I128Hash(s)
}

// flip is an endianness swapper for []byte
func flip(b []byte) []byte {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return b
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

type RemoveAddrReq struct {
	FioAddress      string          `json:"fio_address"`
	PublicAddresses []TokenPubAddr  `json:"public_addresses"`
	MaxFee          uint64          `json:"max_fee"`
	Actor           eos.AccountName `json:"actor"`
	Tpid            string          `json:"tpid"`
}

// NewRemoveAddrReq allows removal of public token/chain addresses
func NewRemoveAddrReq(fioAddress Address, toRemove []TokenPubAddr, actor eos.AccountName) (remove *Action, err error) {
	if !fioAddress.Valid() {
		return nil, errors.New("invalid address")
	}
	if toRemove == nil || len(toRemove) == 0 {
		return nil, errors.New("empty address list supplied")
	}
	return NewAction(
		"fio.address", "remaddress", actor,
		RemoveAddrReq{
			FioAddress:      string(fioAddress),
			PublicAddresses: toRemove,
			MaxFee:          Tokens(GetMaxFee(FeeRemovePubAddress)),
			Actor:           actor,
			Tpid:            CurrentTpid(),
		},
	), nil
}

// RemoveAllAddrReq is for removing all public addresses associated with a FIO address
type RemoveAllAddrReq struct {
	FioAddress string          `json:"fio_address"`
	MaxFee     uint64          `json:"max_fee"`
	Actor      eos.AccountName `json:"actor"`
	Tpid       string          `json:"tpid"`
}

// NewRemoveAllAddrReq allows removal of ALL public token/chain addresses
func NewRemoveAllAddrReq(fioAddress Address, actor eos.AccountName) (remove *Action, err error) {
	if !fioAddress.Valid() {
		return nil, errors.New("invalid address")
	}
	return NewAction(
		"fio.address", "remalladdr", actor,
		RemoveAllAddrReq{
			FioAddress: string(fioAddress),
			MaxFee:     Tokens(GetMaxFee(FeeRemoveAllAddresses)),
			Actor:      actor,
			Tpid:       CurrentTpid(),
		},
	), nil
}

type bundleRemaining struct {
	Bundle int `json:"bundleeligiblecountdown"`
}

// GetBundleRemaining reports on how many free bundled tx remain for an Address
func (api *API) GetBundleRemaining(a Address) (remaining int, err error) {
	if !a.Valid() {
		return 0, errors.New("invalid FIO address")
	}
	hash := I128Hash(string(a))
	gtr, err := api.GetTableRows(eos.GetTableRowsRequest{
		Code:       "fio.address",
		Scope:      "fio.address",
		Table:      "fionames",
		LowerBound: hash,
		UpperBound: hash,
		Limit:      1,
		KeyType:    "i128",
		Index:      "5",
		JSON:       true,
	})
	if err != nil {
		return 0, err
	}
	br := make([]bundleRemaining, 0)
	err = json.Unmarshal(gtr.Rows, &br)
	if err != nil {
		return 0, err
	}
	if br == nil || len(br) != 1 {
		return 0, nil
	}
	return br[0].Bundle, nil
}

type AddBundles struct {
	FioAddress string `json:"fio_address"`
	BundleSets int64 `json:"bundle_sets"`
	MaxFee uint64 `json:"max_fee"`
	Tpid string `json:"tpid"`
	Actor eos.AccountName
}

// NewAddBundles is used to purchase new bundled transactions for an account, 1 bundle set
// is 100 transactions.
func NewAddBundles(fioAddress Address, bundleSets uint64, actor eos.AccountName) (bundles *Action, err error) {
	if !fioAddress.Valid() {
		return nil, errors.New("invalid address")
	}
	return NewAction(
		"fio.address", "addbundles", actor,
		AddBundles{
			FioAddress: string(fioAddress),
			BundleSets: int64(bundleSets),
			MaxFee:     Tokens(GetMaxFee(FeeAddBundles)) * bundleSets,
			Tpid:       CurrentTpid(),
			Actor:      actor,
		},
	), nil
}
