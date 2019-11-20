package fio

import "github.com/eoscanada/eos-go"

type UpdateTpid struct {
	Tpid string `json:"tpid"`
	Owner eos.AccountName `json:"owner"`
	Amount uint64 `json:"amount"`
}

func NewUpdateTpid(actor eos.AccountName, tpid string, amount uint64) *eos.Action {
	return newAction(
		"fio.tpid", "updatepid", actor,
		UpdateTpid{
			Tpid:   tpid,
			Owner:  actor,
			Amount: amount,
		},
	)
}

type RewardsPaid struct {
	Tpid string `json:"tpid"`
}

func NewRewardsPaid(actor eos.AccountName, tpid string) *eos.Action {
	return newAction(
		"fio.tpid", "rewardspaid", actor,
		UpdateTpid{
			Tpid:   tpid,
			Owner:  actor,
		},
	)
}

type UpdateBounty struct {
	Amount uint64 `json:"amount"`
}

func NewUpdateBounty(actor eos.AccountName, amount uint64) *eos.Action {
	return newAction(
		"fio.tpid", "updatebounty", actor,
		UpdateBounty{Amount: amount},
	)
}
