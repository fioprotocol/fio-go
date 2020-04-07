package fio

import "github.com/eoscanada/eos-go"

// PayTpidRewards is a privileged call and not likely to ever be called directly
type PayTpidRewards struct {
	Actor eos.AccountName `json:"actor"`
}

func NewPayTpidRewards(actor eos.AccountName) *Action {
	return NewAction(
		eos.AccountName("fio.treasury"), "tpidclaim", actor,
		PayTpidRewards{Actor: actor},
	)
}

// BpClaim requests payout for a block producer
type BpClaim struct {
	FioAddress string          `json:"fio_address"`
	Actor      eos.AccountName `json:"actor"`
}

func NewBpClaim(fioAddress string, actor eos.AccountName) *Action {
	return NewAction(
		eos.AccountName("fio.treasury"), "bpclaim", actor,
		BpClaim{
			FioAddress: fioAddress,
			Actor:      actor,
		},
	)
}
