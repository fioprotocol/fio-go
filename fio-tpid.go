package fio

import "github.com/eoscanada/eos-go"

// globalTpid is used to store the wallet address for getting rewards (at time of network launch 10% of tx fee).
// This is a global to the package, and only needs to be set once using SetTpid.
var globalTpid string

// SetTpid will set a package variable that will include the provided TPID in all of the calls that support it.
// This only needs to be called once.
func SetTpid(walletAddress string) (ok bool) {
	if ok := Address(walletAddress).Valid(); ok {
		globalTpid = walletAddress
		return true
	}
	return false
}

func CurrentTpid() string {
	return globalTpid
}

type UpdateTpid struct {
	Tpid   string          `json:"tpid"`
	Owner  eos.AccountName `json:"owner"`
	Amount uint64          `json:"amount"`
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
			Tpid:  tpid,
			Owner: actor,
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
