package fio

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	fos "github.com/fioprotocol/fio-go/imports/eos-fio"
	"github.com/fioprotocol/fio-go/imports/eos-fio/fecc"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// VoteProducer votes for a producer
type VoteProducer struct {
	Producers  []string `json:"producers"`
	FioAddress string   `json:"fio_address,omitempty"`
	Actor      fos.AccountName
	MaxFee     uint64 `json:"max_fee"`
}

// NewVoteProducer creates a VoteProducer action: note - fioAddress is optional as of FIP-009
func NewVoteProducer(producers []string, actor fos.AccountName, fioAddress string) *Action {
	sort.Strings(producers)
	return NewAction(
		fos.AccountName("eosio"), "voteproducer", actor,
		VoteProducer{
			Producers:  producers,
			FioAddress: fioAddress,
			Actor:      actor,
			MaxFee:     Tokens(GetMaxFee(FeeVoteProducer)),
		},
	)
}

// BpClaim requests payout for a block producer
type BpClaim struct {
	FioAddress string          `json:"fio_address"`
	Actor      fos.AccountName `json:"actor"`
}

func NewBpClaim(fioAddress string, actor fos.AccountName) *Action {
	return NewAction(
		fos.AccountName("fio.treasury"), "bpclaim", actor,
		BpClaim{
			FioAddress: fioAddress,
			Actor:      actor,
		},
	)
}

// ProducerLocation valid values are 10-80 in increments of 10
type ProducerLocation uint16

const (
	LocationEastAsia         ProducerLocation = 10
	LocationAustralia        ProducerLocation = 20
	LocationWestAsia         ProducerLocation = 30
	LocationAfrica           ProducerLocation = 40
	LocationEurope           ProducerLocation = 50
	LocationEastNorthAmerica ProducerLocation = 60
	LocationSouthAmerica     ProducerLocation = 70
	LocationWestNorthAmerica ProducerLocation = 80
)

type RegProducer struct {
	FioAddress string          `json:"fio_address"`
	FioPubKey  string          `json:"fio_pub_key"`
	Url        string          `json:"url"`
	Location   uint16          `json:"location"`
	Actor      fos.AccountName `json:"actor"`
	MaxFee     uint64          `json:"max_fee"`
}

func NewRegProducer(fioAddress string, fioPubKey string, url string, location ProducerLocation, actor fos.AccountName) (*Action, error) {
	if !strings.HasPrefix(url, "http") {
		return nil, errors.New("url must begin with http:// or https://")
	}
	if !strings.Contains("10 20 30 40 50 60 70 80", strconv.Itoa(int(location))) {
		return nil, errors.New("location must be one of: 10 20 30 40 50 60 70 80")
	}
	return NewAction("eosio", "regproducer", actor,
		RegProducer{
			FioAddress: fioAddress,
			FioPubKey:  fioPubKey,
			Url:        url,
			Location:   uint16(location),
			Actor:      actor,
			MaxFee:     Tokens(GetMaxFee(FeeRegisterProducer)),
		}), nil
}

func MustNewRegProducer(fioAddress string, fioPubKey string, url string, location ProducerLocation, actor fos.AccountName) *Action {
	p, err := NewRegProducer(fioAddress, fioPubKey, url, location, actor)
	if err != nil {
		fmt.Println("MustNewRegProducer failed")
		panic(err)
	}
	return p
}

type UnRegProducer struct {
	FioAddress string          `json:"fio_address"`
	Actor      fos.AccountName `json:"actor"`
	MaxFee     uint64          `json:"max_fee"`
}

func NewUnRegProducer(fioAddress string, actor fos.AccountName) *Action {
	return NewAction("eosio", "unregprod", actor, UnRegProducer{
		FioAddress: fioAddress,
		Actor:      actor,
		MaxFee:     Tokens(GetMaxFee(FeeUnregisterProducer)),
	})
}

type VoteProxy struct {
	Proxy      string          `json:"proxy"`
	FioAddress string          `json:"fio_address,omitempty"`
	Actor      fos.AccountName `json:"actor"`
	MaxFee     uint64          `json:"max_fee"`
}

// NewVoteProxy creates a VoteProxy action: note - fioAddress is optional as of FIP-009
func NewVoteProxy(proxy string, fioAddress string, actor fos.AccountName) *Action {
	return NewAction("eosio", "voteproxy", actor,
		VoteProxy{
			Proxy:      proxy,
			FioAddress: fioAddress,
			Actor:      actor,
			MaxFee:     Tokens(GetMaxFee(FeeProxyVote)),
		},
	)
}

type RegProxy struct {
	FioAddress string          `json:"fio_address"`
	Actor      fos.AccountName `json:"actor"`
	MaxFee     uint64          `json:"max_fee"`
}

func NewRegProxy(fioAddress string, actor fos.AccountName) *Action {
	return NewAction("eosio", "regproxy", actor,
		RegProxy{
			FioAddress: fioAddress,
			Actor:      actor,
			MaxFee:     Tokens(GetMaxFee(FeeRegisterProxy)),
		},
	)
}

type ProducerKey struct {
	AccountName     fos.AccountName `json:"producer_name"`
	BlockSigningKey fecc.PublicKey  `json:"block_signing_key"`
}

type Schedule struct {
	Version   uint32        `json:"version"`
	Producers []ProducerKey `json:"producers"`
}

type ProducerSchedule struct {
	Active   Schedule `json:"active"`
	Pending  Schedule `json:"pending"`
	Proposed Schedule `json:"proposed"`
}

func (api *API) GetProducerSchedule() (*ProducerSchedule, error) {
	res, err := api.HttpClient.Post(api.BaseURL+"/v1/chain/get_producer_schedule", "application/json", bytes.NewReader(nil))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	sched := &ProducerSchedule{}
	err = json.Unmarshal(body, sched)
	if err != nil {
		return nil, err
	}
	return sched, nil
}

// Producers is a modification of the corresponding eos-go structure
type Producers struct {
	Producers               []Producer `json:"producers"`
	TotalProducerVoteWeight string     `json:"total_producer_vote_weight"`
	More                    string     `json:"more"`
}

// Producer is a modification of the corresponding eos-go structure
type Producer struct {
	Owner             fos.AccountName `json:"owner"`
	FioAddress        Address         `json:"fio_address"`
	TotalVotes        string          `json:"total_votes"`
	ProducerPublicKey string          `json:"producer_public_key"`
	IsActive          uint8           `json:"is_active"`
	Url               string          `json:"url"`
	UnpaidBlocks      uint64          `json:"unpaid_blocks"`
	LastClaimTime     string          `json:"last_claim_time"`
	Location          uint8           `json:"location"`
}

// GetFioProducers retrieves the producer table.
// The producers table is a little different on FIO, use this instead of the GetProducers call from eos-go
// TODO: it defaults to a limit of 1,000 ... may want to rethink this as a default
func (api API) GetFioProducers() (fioProducers *Producers, err error) {
	req, err := http.NewRequest("POST", api.BaseURL+`/v1/chain/get_producers`, bytes.NewReader([]byte(`{"limit": 1000}`)))
	if err != nil {
		return nil, err
	}
	req.Header.Add("content-type", "application/json")
	res, err := api.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &fioProducers)
	if err != nil {
		return nil, err
	}
	return
}

type BpJsonOrg struct {
	CandidateName       string `json:"candidate_name"`
	Website             string `json:"website"`
	CodeOfConduct       string `json:"code_of_conduct"`
	OwnershipDisclosure string `json:"ownership_disclosure"`
	Email               string `json:"email"`
	Branding            struct {
		Logo256  string `json:"logo_256"`
		Logo1024 string `json:"logo_1024"`
		LogoSvg  string `json:"logo_svg"`
	} `json:"branding"`
	Location BpJsonLocation `json:"location"`
}

type BpJsonSocial struct {
	Steemit  string `json:"steemit"`
	Twitter  string `json:"twitter"`
	Youtube  string `json:"youtube"`
	Facebook string `json:"facebook"`
	Github   string `json:"github"`
	Reddit   string `json:"reddit"`
	Keybase  string `json:"keybase"`
	Telegram string `json:"telegram"`
	Wechat   string `json:"wechat"`
}

type BpJsonLocation struct {
	Name      string  `json:"name"`
	Country   string  `json:"country"`
	Latitude  float32 `json:"latitude"`
	Longitude float32 `json:"longitude"`
}

type BpJsonNode struct {
	Location     BpJsonLocation `json:"location"`
	NodeType     string         `json:"node_type"`
	P2pEndpoint  string         `json:"p2p_endpoint"`
	BnetEndpoint string         `json:"bnet_endpoint"`
	ApiEndpoint  string         `json:"api_endpoint"`
	SslEndpoint  string         `json:"ssl_endpoint"`
}

type BpJson struct {
	ProducerAccountName string       `json:"producer_account_name"`
	Org                 BpJsonOrg    `json:"org"`
	Nodes               []BpJsonNode `json:"nodes"`
	BpJsonUrl           string       `json:"bp_json_url"`
}

// GetBpJson attempts to retrieve the bp.json file for a producer based on the URL in the eosio.producers table.
// It intentionally rejects URLs that are an IP address, or resolve to a private IP address to reduce the risk of
// SSRF attacks, note however this check is not comprehensive, and is not risk free.
func (api *API) GetBpJson(producer fos.AccountName) (*BpJson, error) {
	return api.getBpJson(producer, false)
}

// allows override of private ip check for tests
func (api *API) getBpJson(producer fos.AccountName, allowIp bool) (*BpJson, error) {
	gtr, err := api.GetTableRows(fos.GetTableRowsRequest{
		Code:       "eosio",
		Scope:      "eosio",
		Table:      "producers",
		LowerBound: string(producer),
		UpperBound: string(producer),
		KeyType:    "name",
		Index:      "4",
		JSON:       true,
	})
	if err != nil {
		return nil, err
	}
	producerRows := make([]Producer, 0)
	err = json.Unmarshal(gtr.Rows, &producerRows)
	if len(producerRows) != 1 {
		return nil, errors.New("account not found in producers table")
	}
	if !strings.HasPrefix(producerRows[0].Url, "http") {
		producerRows[0].Url = "https://" + producerRows[0].Url
	}
	u, err := url.Parse(producerRows[0].Url)
	if err != nil {
		return nil, err
	}
	// ensure this is 1) a hostname, and 2) does not resolve to a private IP range:
	if !allowIp {
		ip := net.ParseIP(u.Host)
		if ip != nil {
			return nil, errors.New("URL is an IP address, refusing to fetch")
		}
		addrs, err := net.LookupHost(u.Host)
		if err != nil {
			return nil, err
		}
		if len(addrs) == 0 {
			return nil, errors.New("could not resolve DNS for url")
		}
		for _, ip := range addrs {
			if isPrivate(net.ParseIP(ip)) {
				return nil, errors.New("url points to a private IP address, refusing to continue")
			}
		}
	}

	var regJson, chainJson string
	info, _ := api.GetInfo()
	if strings.HasSuffix(u.String(), "/") {
		chainJson = u.String() + "bp." + info.ChainID.String() + ".json"
		regJson = u.String() + "bp.json"
	} else {
		chainJson = u.String() + "/" + "bp." + info.ChainID.String() + ".json"
		regJson = u.String() + "/bp.json"
	}

	// try chainId first, ignore error
	resp, err := api.HttpClient.Get(chainJson)
	if err == nil && resp != nil {
		if resp.StatusCode == http.StatusOK {
			body, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if len(body) != 0 {
				bpj := &BpJson{}
				err = json.Unmarshal(body, bpj)
				if err == nil && bpj.ProducerAccountName != "" {
					bpj.BpJsonUrl = chainJson
					return bpj, nil
				}
			}
		}
	}

	resp, err = api.HttpClient.Get(regJson)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	bpj := &BpJson{}
	err = json.Unmarshal(body, bpj)
	if err != nil {
		return nil, err
	}
	if bpj.ProducerAccountName == "" {
		return nil, errors.New("did not get valid bp.json")
	}

	bpj.BpJsonUrl = regJson
	return bpj, nil
}

// adapted from https://github.com/emitter-io/address/blob/master/ipaddr.go
// Copyright (c) 2018 Roman Atachiants
var privateBlocks = [...]*net.IPNet{
	parseCIDR("10.0.0.0/8"),     // RFC 1918 IPv4 private network address
	parseCIDR("100.64.0.0/10"),  // RFC 6598 IPv4 shared address space
	parseCIDR("127.0.0.0/8"),    // RFC 1122 IPv4 loopback address
	parseCIDR("169.254.0.0/16"), // RFC 3927 IPv4 link local address
	parseCIDR("172.16.0.0/12"),  // RFC 1918 IPv4 private network address
	parseCIDR("192.0.0.0/24"),   // RFC 6890 IPv4 IANA address
	parseCIDR("192.0.2.0/24"),   // RFC 5737 IPv4 documentation address
	parseCIDR("192.168.0.0/16"), // RFC 1918 IPv4 private network address
	parseCIDR("::1/128"),        // RFC 1884 IPv6 loopback address
	parseCIDR("fe80::/10"),      // RFC 4291 IPv6 link local addresses
	parseCIDR("fc00::/7"),       // RFC 4193 IPv6 unique local addresses
	parseCIDR("fec0::/10"),      // RFC 1884 IPv6 site-local addresses
	parseCIDR("2001:db8::/32"),  // RFC 3849 IPv6 documentation address
}

func parseCIDR(s string) *net.IPNet {
	_, block, err := net.ParseCIDR(s)
	if err != nil {
		panic(fmt.Sprintf("Bad CIDR %s: %s", s, err))
	}
	return block
}

func isPrivate(ip net.IP) bool {
	if ip == nil {
		return true // presumes a true result gets rejected
	}
	for _, priv := range privateBlocks {
		if priv.Contains(ip) {
			return true
		}
	}
	return false
}

type existVotes struct {
	Producers []string `json:"producers"`
}

type prodRow struct {
	FioAddress string `json:"fio_address"`
}

// GetVotes returns a slice of an account's current votes
func (api *API) GetVotes(account string) (votedFor []string, err error) {
	getVote, err := api.GetTableRows(fos.GetTableRowsRequest{
		Code:  "eosio",
		Scope: "eosio",
		Table: "voters",

		Index:      "3",
		LowerBound: account,
		UpperBound: account,
		Limit:      1,
		KeyType:    "name",
		JSON:       true,
	})
	if err != nil {
		return
	}
	v := make([]*existVotes, 0)
	err = json.Unmarshal(getVote.Rows, &v)
	if err != nil {
		return
	}
	if len(v) == 0 {
		return
	}
	votedFor = make([]string, 0)
	for _, row := range v[0].Producers {
		if row == "" {
			continue
		}
		gtr, err := api.GetTableRows(fos.GetTableRowsRequest{
			Code:       "eosio",
			Scope:      "eosio",
			Table:      "producers",
			LowerBound: row,
			UpperBound: row,
			KeyType:    "name",
			Index:      "4",
			JSON:       true,
		})
		if err != nil {
			continue
		}
		p := make([]*prodRow, 0)
		err = json.Unmarshal(gtr.Rows, &p)
		if err != nil {
			continue
		}
		if len(p) == 1 && p[0].FioAddress != "" {
			votedFor = append(votedFor, p[0].FioAddress)
		}
	}
	return
}
