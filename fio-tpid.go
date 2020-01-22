package fio

import (
	"github.com/eoscanada/eos-go"
	"sync"
)

// globalTpid is used to store the wallet address for getting rewards (at time of network launch 10% of tx fee).
// This is a global to the package, and only needs to be set once using SetTpid.
var globalTpid string
var tpidMux sync.RWMutex

// SetTpid will set a package variable that will include the provided TPID in all of the calls that support it.
// This only needs to be called once.
func SetTpid(walletAddress string) (ok bool) {
	tpidMux.Lock()
	defer tpidMux.Unlock()
	if ok := Address(walletAddress).Valid(); ok {
		globalTpid = walletAddress
		return true
	}
	return false
}

func CurrentTpid() string {
	tpidMux.RLock()
	a := globalTpid
	tpidMux.RUnlock()
	return a
}

type UpdateTpid struct {
	Tpid   string          `json:"tpid"`
	Owner  eos.AccountName `json:"owner"`
	Amount uint64          `json:"amount"`
}

func NewUpdateTpid(actor eos.AccountName, tpid string, amount uint64) *Action {
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

func NewRewardsPaid(actor eos.AccountName, tpid string) *Action {
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

func NewUpdateBounty(actor eos.AccountName, amount uint64) *Action {
	return newAction(
		"fio.tpid", "updatebounty", actor,
		UpdateBounty{Amount: amount},
	)
}
