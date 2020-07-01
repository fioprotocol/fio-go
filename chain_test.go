package fio

import (
	"encoding/json"
	"fmt"
	feos "github.com/fioprotocol/fio-go/imports/eos-fio"
	"math"
	"reflect"
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
	if a[feos.AccountName("eosio")] == nil {
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

	gtr, err := api.GetTableRows(feos.GetTableRowsRequest{
		Code:       "eosio",
		Scope:      "eosio",
		Table:      "producers",
		LowerBound: "2",
		UpperBound: fmt.Sprintf("%d", math.MaxUint32),
		Limit:      1,
		JSON:       true,
	})
	if err != nil {
		t.Error(err)
		return
	}
	prodsAsc := make([]*Producer, 0)
	err = json.Unmarshal(gtr.Rows, &prodsAsc)
	if err != nil {
		t.Error(err)
		return
	}

	gtro, err := api.GetTableRowsOrder(GetTableRowsOrderRequest{
		Code:       "eosio",
		Scope:      "eosio",
		Table:      "producers",
		LowerBound: "2",
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
	if prods[0].FioAddress == prodsAsc[0].FioAddress {
		t.Error("did not get expected result")
		fmt.Println(prods[0].FioAddress, prodsAsc[0].FioAddress)
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
	_, err = api.PushEndpointRaw("/v1/chain/transfer_tokens_pub_key", tx)
	if err != nil {
		t.Error(err)
	}
}

func TestAction_ToEos(t *testing.T) {
	account, _, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	act := NewTransferTokensPubKey(account.Actor, account.PubKey, Tokens(0.0001)).ToEos()
	if reflect.TypeOf(*act).String() != "feos.Action" {
		t.Error("ToEos gave wrong type")
		fmt.Println(reflect.TypeOf(*act).String())
	}
}

func TestNewAction(t *testing.T) {
	a := feos.AccountName("test")

	actOwner := NewActionAsOwner(
		"fio.token", "trnsfiopubky", a,
		TransferTokensPubKey{
			PayeePublicKey: "",
			Amount:         1,
			MaxFee:         Tokens(GetMaxFee(FeeTransferTokensPubKey)),
			Actor:          a,
			Tpid:           CurrentTpid(),
		},
	)
	if actOwner.Authorization[0].Permission != feos.PermissionName("owner") {
		t.Error("NewActionAsOwner did not set owner")
	}

	actPerm := NewActionWithPermission(
		"fio.token", "trnsfiopubky", a, "owner",
		TransferTokensPubKey{
			PayeePublicKey: "",
			Amount:         1,
			MaxFee:         Tokens(GetMaxFee(FeeTransferTokensPubKey)),
			Actor:          a,
			Tpid:           CurrentTpid(),
		},
	)
	if actPerm.Authorization[0].Permission != feos.PermissionName("owner") {
		t.Error("NewActionWitherPermission did not set permission")
	}
}

func TestAPI_GetRefBlock(t *testing.T) {
	_, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	_, _, err = api.GetRefBlock()
	if err != nil {
		t.Error(err)
	}
}

func TestAPI_GetTableByScopeMore(t *testing.T) {
	_, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	res, err := api.GetTableByScopeMore(feos.GetTableByScopeRequest{
		Code:  "eosio",
		Table: "producers",
		Limit: 1,
	})
	if err != nil {
		t.Error(err)
		return
	}
	if res.More != true {
		t.Error("expected more records")
	}
}
