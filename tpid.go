package fio

import (
	"github.com/fioprotocol/fio-go/imports/eos-fio"
	"sync"
)

// globalTpid is used to store the wallet address for getting rewards (at time of network launch 10% of tx fee).
// This is a global to the package, and only needs to be set once using SetTpid.
var globalTpid string
var tpidMux sync.RWMutex

// SetTpid will set a package variable that will include the provided TPID in all of the calls that support it.
// This only needs to be called once. By default it is empty, and is recommended for wallet providers or other
// service providers to set at initialization via SetTpid to get rewards.
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

// PayTpidRewards is used for wallets "technology provided id" to claim incentive rewards
type PayTpidRewards struct {
	Actor eos.AccountName `json:"actor"`
}

func NewPayTpidRewards(actor eos.AccountName) *Action {
	return NewAction(
		eos.AccountName("fio.treasury"), "tpidclaim", actor,
		PayTpidRewards{Actor: actor},
	)
}

// UpdateTpid is a privileged call
type UpdateTpid struct {
	Tpid   string          `json:"tpid"`
	Owner  eos.AccountName `json:"owner"`
	Amount uint64          `json:"amount"`
}

func NewUpdateTpid(actor eos.AccountName, tpid string, amount uint64) *Action {
	return NewAction(
		"fio.tpid", "updatepid", actor,
		UpdateTpid{
			Tpid:   tpid,
			Owner:  actor,
			Amount: amount,
		},
	)
}

// RewardsPaid is privileged
type RewardsPaid struct {
	Tpid string `json:"tpid"`
}

func NewRewardsPaid(actor eos.AccountName, tpid string) *Action {
	return NewAction(
		"fio.tpid", "rewardspaid", actor,
		UpdateTpid{
			Tpid:  tpid,
			Owner: actor,
		},
	)
}

// UpdateBounty is privileged
type UpdateBounty struct {
	Amount uint64 `json:"amount"`
}

func NewUpdateBounty(actor eos.AccountName, amount uint64) *Action {
	return NewAction(
		"fio.tpid", "updatebounty", actor,
		UpdateBounty{Amount: amount},
	)
}
