package fio

import (
	"errors"
	"github.com/eoscanada/eos-go"
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

// NewTransferTokensPubKey builds an eos.Action for sending FIO tokens
func NewVoteProducer(producers []string, actor eos.AccountName, fioAddress string) *Action {
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
