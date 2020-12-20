package fio

import (
	"testing"
	"time"
)

func TestNewTransferLockedTokens(t *testing.T) {
	acc, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	random, _ := NewRandomAccount()
	_, err = api.NewValidTransferLockedTokens(acc.Actor, random.PubKey, true, []LockPeriods{{Duration: 86400, Percent: 100}}, Tokens(10.0))
	if err == nil {
		t.Error("NewValidTransferLockedTokens allowed allocation < 100%")
	}
	_, err = api.NewValidTransferLockedTokens(acc.Actor, acc.PubKey, true, []LockPeriods{{Duration: 86400, Percent: 100}}, Tokens(100.0))
	if err == nil {
		t.Error("NewValidTransferLockedTokens allowed transfer to existing account")
	}

	_, err = api.SignPushActions(NewTransferLockedTokens(acc.Actor, random.PubKey, true, []LockPeriods{{Duration: 86400, Percent: 100.0}}, Tokens(100.0)))
	if err != nil {
		t.Error(err)
		return
	}

	bal, err := api.GetFioBalance(random.PubKey)
	if err != nil {
		t.Error(err)
	}
	if bal.Balance != 100 * 1_000_000_000 || bal.Available != 0 {
		t.Error("could not confirm tokens were locked in random account's balance")
	}
}

func TestGenesisLockedTokens_ActualRemaining(t *testing.T) {
	lt := GenesisLockedTokens{
		Name:                  "aaaaaa",
		TotalGrantAmount:      1000,
		UnlockedPeriodCount:   0,
		GrantType:             LockedFounder,
		InhibitUnlocking:      0,
		RemainingLockedAmount: 1000,
		Timestamp:             uint32(time.Now().Add(time.Duration(-91*24) * time.Hour).UTC().Unix()),
	}
	rem, err := lt.ActualRemaining()
	if err != nil {
		t.Error(err)
	}
	if rem != 1000 - 60 {
		t.Error("invalid calculation for first unlock")
	}
	lt.Timestamp = uint32(time.Now().Add(time.Duration(-271*24) * time.Hour).UTC().Unix())
	rem, err = lt.ActualRemaining()
	if err != nil {
		t.Error(err)
	}
	if rem != 1000 - 248 {
		t.Error("invalid calculation for second unlock")
	}

	lt.GrantType = LockedMember
	lt.InhibitUnlocking = 1
	rem, err = lt.ActualRemaining()
	if err != nil {
		t.Error(err)
	}
	if rem != 1000 {
		t.Error("invalid calculation for inhibited tokens")
	}
}

/*
func TestAPI_GetCirculatingSupply(t *testing.T) {
	api, _, err := NewConnection(nil, "http://mainnet:8887")
	if err != nil {
		t.Error(err)
		return
	}
	circ, minted, locked, err := api.GetCirculatingSupply()
	if err != nil {
		t.Error(err)
		return
	}
	const b uint64 = 1_000_000_000
	pp := message.NewPrinter(language.AmericanEnglish)
	pp.Printf("circulating %d, minted %d, locked %d\n", circ/b, minted/b, locked/b)
}
*/