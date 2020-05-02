package fio

import (
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestTpid(t *testing.T) {
	if ok := SetTpid("bad@address@shouldfail"); ok {
		t.Error("should not be able to set invalid tpid")
	}

	current := CurrentTpid()
	if ok := SetTpid("adam@dapixdev"); !ok {
		t.Error("could not set new tpid")
	}
	if current == CurrentTpid() {
		t.Error("tpid did not change")
	}

	_, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	account, err := NewAccountFromWif("5KQ6f9ZgUtagD3LZ4wcMKhhvK9qy4BuwL3L1pkm6E2v62HCne2R")
	if err != nil {
		t.Error(err)
		return
	}
	var opts = &TxOptions{}
	api, opts, err = NewConnection(account.KeyBag, api.BaseURL)
	if err != nil {
		t.Error(err)
		return
	}
	// this might fail since there aren't likely any rewards, so first try to generate some:
	faucet, fApi, fOpts, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	rand.Seed(time.Now().UnixNano())
	for i:=0; i < 10; i++ {
		randomAccount, err := NewRandomAccount()
		if err != nil {
			t.Error(err)
			break
		}
		_, err = fApi.SignPushTransaction(NewTransaction(
			[]*Action{NewRegDomain(faucet.Actor, word(), randomAccount.PubKey)},
			fOpts,
		), fOpts.ChainID, CompressionNone)
		if err != nil {
			t.Error(err)
			break
		}
	}
	_, packed, err := api.SignTransaction(NewTransaction(
		[]*Action{NewPayTpidRewards(account.Actor)}, opts),
		opts.ChainID, CompressionNone,
	)
	if err != nil {
		t.Error(err)
		return
	}
	j, err := api.PushTransactionRaw(packed)
	if err != nil && !strings.Contains(err.Error(), "An invalid request was sent in") {
		t.Error(err)
	} else if !strings.Contains(string(j), "tpids_paid") {
		t.Error("expected tpid payout: "+string(j))
	}
}

func word() string {
	var w string
	for i := 0; i < 8; i++ {
		w = w + string(byte(rand.Intn(26)+97))
	}
	return w
}