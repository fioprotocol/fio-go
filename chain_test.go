package fio

import (
	"encoding/json"
	"fmt"
	"github.com/eoscanada/eos-go"
	"math"
	"testing"
)

func TestAPI_GetCurrentBlock(t *testing.T) {
	_, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	b := api.GetCurrentBlock()
	if b == 0 {
		t.Error("could not fetch latest block")
	}
}

func TestAPI_AllABIs(t *testing.T) {
	_, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	a, err := api.AllABIs()
	if err != nil {
		t.Error(err)
		return
	}
	if len(a) == 0 {
		t.Error("did not get abis")
		return
	}
	if a[eos.AccountName("eosio")] == nil {
		t.Error("did not get abi for eosio")
		return
	}
}

func TestAPI_GetTableRowsOrder(t *testing.T) {
	_, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	if api.BaseURL != "http://dev:8889" {
		// this will only work on a dev machine with known producer list
		return
	}

	gtro, err := api.GetTableRowsOrder(GetTableRowsOrderRequest{
		Code:       "eosio",
		Scope:      "eosio",
		Table:      "producers",
		LowerBound: "0",
		UpperBound: fmt.Sprintf("%d", math.MaxUint32),
		Limit:      1,
		JSON:       true,
		Reverse:    true,
	})
	if err != nil {
		t.Error(err)
		return
	}
	prods := make([]*Producer, 0)
	err = json.Unmarshal(gtro.Rows, &prods)
	if err != nil {
		t.Error(err)
		return
	}

	if prods == nil || len(prods) == 0 {
		t.Error("did not get a query result")
		return
	}
	if prods[0].FioAddress != Address("bp3@dapixdev") {
		t.Error("did not get expected result")
	}
}

func TestAPI_GetRefBlockFor(t *testing.T) {
	rbn, rbp, err := GetRefBlockFor(27309, "00006aadc588b4c4d14cb0a0f677b7741746ae0dd3bb68e42a21c36c6d94ecb4")
	if err != nil {
		t.Error(err)
		return
	}
	if rbn != 27309 || rbp != 2695908561 {
		t.Error("did not get expected prefix or id")
	}
}

func TestAPI_GetBlockHeaderState(t *testing.T) {
	_, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	info, err := api.GetInfo()
	if err != nil {
		t.Error(err)
		return
	}
	bhs, err := api.GetBlockHeaderState(info.HeadBlockNum)
	if err != nil {
		t.Error(err)
		return
	}
	if bhs == nil {
		t.Error("got nil result")
		return
	}
	if bhs.BlockNum != info.HeadBlockNum {
		t.Error("did not get expected result")
	}

	ok, last := bhs.ProducerToLast(ProducerToLastImplied)
	if !ok {
		t.Error("last implied not found")
	}
	if len(last) < 3 {
		t.Error("last implied was missing producers")
	}

	ok, last = bhs.ProducerToLast(ProducerToLastProduced)
	if !ok {
		t.Error("last produced not found")
	}
	if len(last) < 3 {
		t.Error("last produced was missing producers")
	}

}

func TestAPI_PushEndpointRaw(t *testing.T) {
	account, api, opts, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}

	randAccount, _ := NewRandomAccount()
	_, tx, err := api.SignTransaction(
		NewTransaction(
			[]*Action{
				NewTransferTokensPubKey(account.Actor, randAccount.PubKey, Tokens(0.0001))},
			opts),
		opts.ChainID, CompressionNone,
	)
	_, err = api.PushEndpointRaw("transfer_tokens_pub_key", tx)
	if err != nil {
		t.Error(err)
	}
}

