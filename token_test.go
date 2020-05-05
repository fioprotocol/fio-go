package fio

import "testing"

func TestFioToken(t *testing.T) {
	account, api, opts, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	myBal, err := api.GetBalance(account.Actor)
	if err != nil {
		t.Error(err)
		return
	}
	if myBal < 1_000_000.0 {
		t.Error("do not have tokens for test")
		return
	}

	randomAccount, err := NewRandomAccount()
	if err != nil {
		t.Error(err)
		return
	}

	_, err = api.SignPushTransaction(NewTransaction(
		[]*Action{NewTransferTokensPubKey(account.Actor, randomAccount.PubKey, Tokens(1.0))},
		opts,
	), opts.ChainID, CompressionNone)
	if err != nil {
		t.Error(err)
		return
	}

	received, err := api.GetBalance(randomAccount.Actor)
	if err != nil {
		t.Error(err)
		return
	}
	if received != 1.0 {
		t.Error("balance was wrong")
	}
}
