package fio

import (
	"encoding/json"
	"errors"
	"github.com/fioprotocol/fio-go/eos"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

/*
   Genesis lock types
*/

// set as variables to allow override on development nets
var (
	LockedInitial              = 90    // initial days before first unlock period
	LockedInitialPct   float64 = 0.06  // percentage unlocked after first period
	LockedIncrement            = 180   // each additional unlock period, 2nd unlock = LockedInitial + LockedIncrement, 3rd = LockedInitial + (2 * LockedIncrement), etc
	LockedIncrementPct float64 = 0.188 // percent unlocked each additional period: 1st = 6%, 2nd = 24.8% etc.
	LockedPeriods              = 6     // number of unlock periods
)

const (
	LockedFounder  uint32 = iota + 1 // founder tokens: cannot vote until unlocked, can use for fees.
	LockedMember                     // foundation member (wallets/exchanges) incentives: if inhibit unlocking is set, cannot be used, votable until 2nd unlock or if uninhibited.
	LockedPresale                    // presale tokens, can vote while locked
	LockedGiveaway                   // foundation tokens used for giveaways, can only be used to register addresses
)

// GenesisLockedTokens holds information about tokens that were locked at chain genesis.
type GenesisLockedTokens struct {
	Name                  eos.AccountName `json:"name"`
	TotalGrantAmount      uint64          `json:"total_grant_amount"`
	UnlockedPeriodCount   uint32          `json:"unlocked_period_count"`
	GrantType             uint32          `json:"grant_type"`
	InhibitUnlocking      uint32          `json:"inhibit_unlocking"`
	RemainingLockedAmount uint64          `json:"remaining_locked_amount"` // Do not trust on-chain value, only calculated on activity.
	Timestamp             uint32          `json:"timestamp"`
}

// GetGenesisLockedTokens gives details about the locked tokens for a specific account
func (api *API) GetGenesisLockedTokens(accountOrPubkey string) (hasLocked bool, locked *GenesisLockedTokens, err error) {
	var actor eos.AccountName
	switch len(accountOrPubkey) {
	case 12:
		actor = eos.AccountName(accountOrPubkey)
	case 53:
		actor, err = ActorFromPub(accountOrPubkey)
		if err != nil {
			return false, nil, err
		}
	default:
		return false, nil, errors.New("invalid account or pubkey provided for looking up locked tokens")
	}
	gtr, err := api.GetTableRows(eos.GetTableRowsRequest{
		Code:       "eosio",
		Scope:      "eosio",
		Table:      "lockedtokens",
		LowerBound: string(actor),
		UpperBound: string(actor),
		Limit:      1,
		KeyType:    "name",
		Index:      "1",
		JSON:       true,
	})
	if err != nil {
		return false, nil, err
	}
	lts := make([]GenesisLockedTokens, 0)
	err = json.Unmarshal(gtr.Rows, &lts)
	if err != nil {
		return false, nil, err
	}
	if len(lts) == 0 {
		return false, nil, nil
	}
	return true, &lts[0], nil
}

// ActualRemaining calculates the outstanding locked tokens, this is needed because RemainingLockedAmount is only updated
// when an account spends or receives tokens. This should be subtracted from the current balance to calculate available
// tokens.
func (g *GenesisLockedTokens) ActualRemaining() (tokens uint64, err error) {
	lockStart := time.Unix(int64(g.Timestamp), 0).UTC()
	switch g.GrantType {
	case LockedGiveaway:
		// every transaction will cause RemainingLockedAmount to get calculated, so it is always accurate.
		return g.RemainingLockedAmount, nil

	case LockedMember:
		// Members had until the 2nd unlock to integrate FIO into their wallet or exchange, if it was done the tokens
		// would have the InhibitUnlock set to 0, meaning the tokens continue to unlock as scheduled.
		if g.InhibitUnlocking == 1 {
			// they may have spent fees before the tokens were locked forever, so total_grant_amount could too high
			return g.RemainingLockedAmount, nil
		}
		// otherwise same unlock schedule applies
		fallthrough

	case LockedFounder, LockedPresale:
		days := int(time.Now().Sub(lockStart).Hours()) / 24
		// have not passed first period
		if days <= LockedInitial {
			return g.RemainingLockedAmount, nil
		}
		// first unlock passed
		pct := LockedInitialPct
		// add percentage for each additional
		for i := 1; i <= LockedPeriods; i++ {
			if days > (i*LockedIncrement)+LockedInitial {
				pct += LockedIncrementPct
			} else {
				break
			}
		}
		unlocked := uint64(math.Round(float64(g.TotalGrantAmount) * pct))
		if g.RemainingLockedAmount < (g.TotalGrantAmount - unlocked) {
			return g.RemainingLockedAmount, nil
		}
		return g.TotalGrantAmount - unlocked, nil

	default:
		return 0, errors.New("unknown token lock type")
	}
}

// GetTotalGenesisLockTokens tallies the remaining locked tokens based upon the values in the lockedtokens table
func (api *API) GetTotalGenesisLockTokens() (total uint64, founder uint64, member uint64, presale uint64, giveaway uint64, err error) {
	nameQueries := splitNames(10)
	for i := range nameQueries {
		gtr, err := api.GetTableRows(eos.GetTableRowsRequest{
			Code:       "eosio",
			Scope:      "eosio",
			Table:      "lockedtokens",
			LowerBound: nameQueries[i].LowerName,
			UpperBound: nameQueries[i].UpperName,
			KeyType:    "name",
			Index:      "1",
			JSON:       true,
		})
		if err != nil {
			return 0, 0, 0, 0, 0, err
		}
		rows := make([]GenesisLockedTokens, 0)
		err = json.Unmarshal(gtr.Rows, &rows)
		if err != nil {
			return 0, 0, 0, 0, 0, err
		}
		var locked uint64
		for j := range rows {
			locked, err = rows[j].ActualRemaining()
			if err != nil {
				return 0, 0, 0, 0, 0, err
			}
			total += locked
			switch rows[j].GrantType {
			case LockedFounder:
				founder += locked
			case LockedMember:
				member += locked
			case LockedPresale:
				presale += locked
			case LockedGiveaway:
				giveaway += locked
			}
		}
	}
	return
}

type nameRange struct {
	LowerI64  uint64
	LowerName string
	UpperI64  uint64
	UpperName string
}

// splitNames is a handy helper for looping over name primary indexes, while not efficient there isn't
// any way to know how many records there are, and this will break it into smaller chunks
func splitNames(size int) []nameRange {
	names := make([]nameRange, 0)
	iter := math.MaxUint64 / uint64(size)
	var high, low uint64
	for s := size; s > 0; s -= 1 {
		if s == size {
			high, _ = eos.StringToName("zzzzzzzzzzzz") // highest name is actually 615 less than max uint64
		} else {
			high = uint64(s) * iter
		}
		if s == 1 || high < iter {
			low = 1 // can't be 0
		} else {
			low = high - iter + 1
		}
		names = append(names, nameRange{
			LowerI64:  low,
			LowerName: eos.NameToString(low),
			UpperI64:  high,
			UpperName: eos.NameToString(high),
		})
	}
	sort.Slice(names, func(i, j int) bool {
		return names[i].LowerI64 < names[j].LowerI64
	})
	return names
}

/*
   FIP6 lock types and actions: https://github.com/fioprotocol/fips/blob/master/fip-0006.md
*/

// LockPeriods specifies how long a portion of the locked tokens will be locked by seconds and percentage, a slice
// of these is provided when locking tokens and must total 100%. It's also important to know that when staking tokens
// only certain durations will be valid, see https://github.com/fioprotocol/fips/blob/master/fip-0021.md
type LockPeriods struct {
	Duration uint64  `json:"duration"`
	Percent  float64 `json:"percent"`
}

const (
	CanVoteNone int32 = iota
	CanVoteAll
)

type TransferLockedTokens struct {
	PayeePublicKey string          `json:"payee_public_key"`
	CanVote        int32           `json:"can_vote"`
	Periods        []LockPeriods   `json:"periods"`
	Amount         uint64          `json:"amount"`  // differs from ABI, not sure why this call is unsigned, but making consistent in SDK.
	MaxFee         uint64          `json:"max_fee"` // ABI also differs here.
	Actor          eos.AccountName `json:"actor"`
	Tpid           string          `json:"tpid"`
}

// NewTransferLockedTokens creates an action used to transfer locked tokens to an account. This must be a new account, and cannot have existing tokens or addresses.
func NewTransferLockedTokens(actor eos.AccountName, recipientPubKey string, canVote bool, periods []LockPeriods, amount uint64) *Action {
	can := CanVoteNone
	if canVote {
		can = CanVoteAll
	}
	return NewAction(
		"fio.token", "trnsloctoks", actor,
		TransferLockedTokens{
			PayeePublicKey: recipientPubKey,
			CanVote:        can,
			Periods:        periods,
			Amount:         amount,
			MaxFee:         Tokens(GetMaxFee(FeeTransferLockedTokens)),
			Actor:          actor,
			Tpid:           CurrentTpid(),
		},
	)
}

// NewValidTransferLockedTokens is the same as NewTransferLockedTokens, but adds checks to ensure the account does not exist, and the periods are legit
func (api *API) NewValidTransferLockedTokens(actor eos.AccountName, recipientPubKey string, canVote bool, periods []LockPeriods, amount uint64) (*Action, error) {
	can := CanVoteNone
	if canVote {
		can = CanVoteAll
	}
	tlt := TransferLockedTokens{
		PayeePublicKey: recipientPubKey,
		CanVote:        can,
		Periods:        periods,
		Amount:         amount,
		MaxFee:         Tokens(GetMaxFee(FeeTransferLockedTokens)),
		Actor:          actor,
		Tpid:           CurrentTpid(),
	}
	if e := tlt.valid(api); e != nil {
		return nil, e
	}
	return NewAction("fio.token", "trnsloctoks", actor, tlt), nil
}

func (tlt *TransferLockedTokens) valid(api *API) error {
	switch true {
	case eos.CheckUnderOver(tlt.MaxFee) != nil:
		return eos.CheckUnderOver(tlt.MaxFee)
	case eos.CheckUnderOver(tlt.Amount) != nil:
		return eos.CheckUnderOver(tlt.Amount)
	case tlt.Amount == 0:
		return errors.New("must transfer a positive amount")
	case tlt.CanVote > CanVoteAll:
		return errors.New("invalid value for can_vote, must be 0 or 1")
	case len(tlt.Periods) == 0:
		return errors.New("no periods provided")
	}
	act, err := ActorFromPub(tlt.PayeePublicKey)
	if err != nil {
		return err
	}
	var pct float64
	for i := range tlt.Periods {
		pct += tlt.Periods[i].Percent
	}
	if pct != 100 {
		return errors.New("percentage must equal 100%")
	}
	resp, _ := api.GetFioAccount(string(act))
	if resp != nil {
		return errors.New("cannot transfer to an existing account")
	}
	return nil
}

type LockTokensResp struct {
	Id                  uint32          `json:"id"`
	OwnerAccount        eos.AccountName `json:"owner_account"`
	LockAmount          uint64          `json:"lock_amount"`
	PayoutsPerformed    uint32          `json:"payouts_performed"`
	CanVote             int32           `json:"can_vote"`
	Periods             []*LockPeriods  `json:"periods"`
	RemainingLockAmount uint64          `json:"remaining_lock_amount"`
	TimeStamp           int64           `json:"time_stamp"`
}

// GetTotalLockTokens provides the total number of (FIP6) locked tokens by iterating through the locktokens table.
func (api *API) GetTotalLockTokens() (uint64, error) {
	var total uint64
	var more bool
	const iter int64 = 100
	now := time.Now().UTC().Unix()
	for i := int64(0); !more; i += iter {
		gtr, err := api.GetTableRows(eos.GetTableRowsRequest{
			Code:       "eosio",
			Scope:      "eosio",
			Table:      "locktokens",
			LowerBound: strconv.FormatInt(i, 10),
			Limit:      uint32(iter - 1),
			KeyType:    "i64",
			Index:      "1",
			JSON:       true,
		})
		if err != nil {
			if err.Error() == "Internal Service Error - (fc): Contract Table Query Exception: Table locktokens is not specified in the ABI" {
				// this may be running before FIP6 has been deployed
				return 0, nil
			}
			return 0, err
		}
		ltr := make([]LockTokensResp, 0)
		err = json.Unmarshal(gtr.Rows, &ltr)
		if err != nil {
			return 0, err
		}
		for i := range ltr {
			total += ltr[i].LockAmount
			for j := range ltr[i].Periods {
				if int64(ltr[i].Periods[j].Duration)+ltr[i].TimeStamp < now {
					total -= ltr[i].LockAmount - uint64((math.Round(ltr[i].Periods[j].Percent*100.0)*float64(ltr[i].LockAmount))/10000)
				}
			}
		}
		if !gtr.More {
			more = false
		}
	}
	return total, nil
}

/*
   Circulating Supply
*/

// GetCirculatingSupply returns the number of spendable tokens based on current supply - genesis locks - locked tokens
// this is a very busy call, requiring multiple requests to calculate the result, and it is recommended to cache the
// output if needed frequently.
func (api *API) GetCirculatingSupply() (circulating uint64, minted uint64, locked uint64, err error) {
	var supply, genesis, userLock, rewLock uint64
	gcr := &eos.GetCurrencyStatsResp{}
	gcr, err = api.GetCurrencyStats("fio.token", "FIO")
	if err != nil {
		return
	}
	// not getting valid values from gcr.Supply.ToUint64()
	sf, err := strconv.ParseFloat(strings.Split(gcr.Supply.String(), " ")[0], 64)
	if err != nil {
		return
	}
	supply = uint64(sf * 1_000_000_000.0)
	genesis, _, _, _, _, err = api.GetTotalGenesisLockTokens()
	if err != nil {
		return
	}
	userLock, err = api.GetTotalLockTokens()
	if err != nil {
		return
	}
	rewLock, err = api.GetLockedBpRewards()
	if err != nil {
		return
	}
	return supply - genesis - userLock - rewLock, supply, genesis + userLock + rewLock, nil
}

// GetLockedBpRewards gets the unpaid rewards for block producers: Fees collected for FIO Address/Domain
// registration are not immediately distributed, but rather locked and distributed evenly every day over
// a period of one year
func (api *API) GetLockedBpRewards() (locked uint64, err error) {
	gtr, err := api.GetTableRows(eos.GetTableRowsRequest{
		Code:  "fio.treasury",
		Scope: "fio.treasury",
		Table: "bpbucketpool",
		JSON:  true,
	})
	if err != nil {
		return
	}
	type lockedRew struct {
		Rewards uint64 `json:"rewards"`
	}
	rows := make([]lockedRew, 0)
	err = json.Unmarshal(gtr.Rows, &rows)
	if err != nil {
		return
	}
	if len(rows) == 0 {
		return 0, errors.New("unknown error getting bpbucketpool")
	}
	return rows[0].Rewards, nil
}
