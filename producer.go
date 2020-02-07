package fio

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eoscanada/eos-go"
	"io/ioutil"
	"sort"
	"strconv"
	"strings"
)

/*
- name: voteproducer
  base: ""
  fields:
    - name: producers
      type: string[]
    - name: fio_address
      type: string
    - name: actor
      type: name
    - name: max_fee
      type: int64
*/
type VoteProducer struct {
	Producers  []string `json:"producers"`
	FioAddress string   `json:"fio_address"`
	Actor      eos.AccountName
	MaxFee     uint64 `json:"max_fee"`
}

func NewVoteProducer(producers []string, actor eos.AccountName, fioAddress string) *Action {
	sort.Strings(producers)
	return NewAction(
		eos.AccountName("eosio"), "voteproducer", actor,
		VoteProducer{
			Producers:  producers,
			FioAddress: fioAddress,
			Actor:      actor,
			MaxFee:     Tokens(GetMaxFee(FeeVoteProducer)),
		},
	)
}

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
	Actor      eos.AccountName `json:"actor"`
	MaxFee     uint64          `json:"max_fee"`
}

func NewRegProducer(fioAddress string, fioPubKey string, url string, location ProducerLocation, actor eos.AccountName) (*Action, error) {
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

func MustNewRegProducer(fioAddress string, fioPubKey string, url string, location ProducerLocation, actor eos.AccountName) *Action {
	p, err := NewRegProducer(fioAddress, fioPubKey, url, location, actor)
	if err != nil {
		fmt.Println("MustNewRegProducer failed")
		panic(err)
	}
	return p
}

type UnRegProducer struct {
	FioAddress string          `json:"fio_address"`
	Actor      eos.AccountName `json:"actor"`
	MaxFee     uint64          `json:"max_fee"`
}

func NewUnRegProducer(fioAddress string, actor eos.AccountName) *Action {
	return NewAction("eosio", "unregprod", actor, UnRegProducer{
		FioAddress: fioAddress,
		Actor:      actor,
		MaxFee:     Tokens(GetMaxFee(FeeUnregisterProducer)),
	})
}

type VoteProxy struct {
	Proxy string `json:"proxy"`
	FioAddress string `json:"fio_address"`
	Actor eos.AccountName `json:"actor"`
	MaxFee uint64 `json:"max_fee"`
}

func NewVoteProxy(proxy string, fioAddress string, actor eos.AccountName) *Action {
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
	FioAddress string `json:"fio_address"`
	Actor eos.AccountName `json:"actor"`
	MaxFee uint64 `json:"max_fee"`
}

func NewRegProxy(fioAddress string, actor eos.AccountName) *Action {
	return NewAction("eosio", "regproxy", actor,
		RegProxy{
			FioAddress: fioAddress,
			Actor:      actor,
			MaxFee:     Tokens(GetMaxFee(FeeRegisterProxy)),
		},
	)
}

// Schedule is a convenience struct for deserializing the producer schedule, it is fully
// encapsulated to prevent conflicts with other types
type ProducerSchedule struct {
	Active struct {
		Version uint32 `json:"version"`
		Producers []struct {
			ProducerName eos.AccountName `json:"producer_name"`
			BlockSigningKey string `json:"block_signing_key"`
		} `json:"producers"`
	} `json:"active"`
	Pending struct {
		Version uint32 `json:"version"`
		Producers []struct {
			ProducerName eos.AccountName `json:"producer_name"`
			BlockSigningKey string `json:"block_signing_key"`
		} `json:"producers"`
	} `json:"pending"`
	Proposed struct {
		Version uint32 `json:"version"`
		Producers []struct {
			ProducerName eos.AccountName `json:"producer_name"`
			BlockSigningKey string `json:"block_signing_key"`
		} `json:"producers"`
	}
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

