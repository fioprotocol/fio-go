package fio

import "github.com/eoscanada/eos-go"

type CreateFee struct {
	EndPoint  string `json:"end_point"`
	Type      uint64 `json:"type"`
	SufAmount uint64 `json:"suf_amount"`
}

type FeeValue struct {
	EndPoint string `json:"end_point"`
	Value    uint64 `json:"value"`
}

type SetFeeVote struct {
	FeeRatios []FeeValue `json:"fee_ratios"`
	Actor     string     `json:"actor"`
}

type BundleVote struct {
	BundledTransactions float64 `json:"bundled_transactions"`
	Actor               string  `json:"actor"`
}

type SetFeeMult struct {
	Multiplier float64 `json:"multiplier"`
	Actor      string  `json:"actor"`
}

type FioFee struct {
	FeeId        uint64      `json:"fee_id"`
	EndPoint     string      `json:"end_point"`
	EndPointHash eos.Uint128 `json:"end_point_hash"`
	Type         uint64      `json:"type"`
	SufAmount    uint64      `json:"suf_amount"`
}

type FeeVoter struct {
	BlockProducerName eos.AccountName `json:"block_producer_name"`
	FeeMultiplier     float64         `json:"fee_multiplier"`
	LastVoteTimestamp uint64          `json:"lastvotetimestamp"`
}

type FeeVote struct {
	Id                uint64          `json:"id"`
	BlockProducerName eos.AccountName `json:"block_producer_name"`
	EndPoint          string          `json:"end_point"`
	EndPointHash      uint64          `json:"end_point_hash"`
	SufAmount         uint64          `json:"suf_amount"`
	LastVoteTimestamp uint64          `json:"lastvotetimestamp"`
}

type BundleVoter struct {
	BlockProducerName eos.AccountName `json:"block_producer_name"`
	BundleVoteNumber  uint64          `json:"bundlevotenumber"`
	LastVoteTimestamp uint64          `json:"lastvotetimestamp"`
}
