package fio

import (
	"encoding/json"
	"fmt"
	"github.com/fioprotocol/fio-go/eos"
	"math"
	"strconv"
	"testing"
)

func TestUpdateMaxFees(t *testing.T) {
	_, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	// force the fee to be wrong, has to be done after connecting
	maxFees["add_pub_address"] = 0.0
	if ok := api.RefreshFees(); !ok {
		t.Error("could not update fees")
	}
	if maxFees["add_pub_address"] == 0.0 {
		t.Error("did not update")
	}
}

func TestAPI_GetFee(t *testing.T) {
	account, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	_, _, err = account.GetNames(api)
	if err != nil || len(account.Addresses) == 0 {
		t.Error("account should have an address")
		return
	}

	// tests both max fee and get fee
	max := GetMaxFee(FeeAddPubAddress)
	if max == 0 || max != GetMaxFeeByAction("addaddress") {
		t.Error("got wrong max fee")
	}
	actual, err := api.GetFee(account.Addresses[0].FioAddress, FeeAddPubAddress)
	if actual != 0 {
		t.Error("fee should have been bundled")
	}
}

func Test_NewSetFeeVote(t *testing.T) {
	acc, api, opts, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}

	opts.Compress = CompressionZlib
	resp, err := api.SignPushActionsWithOpts([]*eos.Action{
		NewSetFeeVote([]*FeeValue{
			{
				EndPoint: "register_fio_domain",
				Value:    40000000000,
			},
		}, acc.Actor).ToEos(),
	}, &opts.TxOptions)
	if err != nil {
		t.Error(err)
		fmt.Println(resp)
	}
}

func Test_FeeOverUnderflow(t *testing.T) {
	account, api, opts, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}

	// Baseline
	goodFee := NewAction("fio.address", "rmalladdr", account.Actor, RemoveAllAddrReq{
		FioAddress: "test@123",
		MaxFee: uint64(math.MaxInt64),
		Actor: account.Actor,
		Tpid: "123@test",
	})
	_, _, err = api.SignTransaction(NewTransaction([]*Action{goodFee}, opts), opts.ChainID, CompressionNone)
	if err != nil {
		t.Error("legit fee blocked when signing transaction")
	}

	// negative int into uint
	// it's actually a little difficult to trick the compiler into allowing this to be triggered, so very low risk.
	var temp interface{}
	temp = int64(-1)
	underFlowFee := NewAction("fio.address", "rmalladdr", account.Actor, RemoveAllAddrReq{
		FioAddress: "test@123",
		// use reflection to forcefully cast, results in MaxFee:18446744073709551615
		MaxFee: uint64(temp.(int64)), // #nosec
		Actor: account.Actor,
		Tpid: "123@test",
	})
	_, _, err = api.SignTransaction(NewTransaction([]*Action{underFlowFee}, opts), opts.ChainID, CompressionNone)
	if err == nil {
		t.Error("potential uint underflow not blocked when signing transaction")
	}

	// uint too large for int
	// this, however, could happen easily.
	overFlowFee := NewAction("fio.address", "rmalladdr", account.Actor, RemoveAllAddrReq{
		FioAddress: "test@123",
		MaxFee: uint64(math.MaxInt64 + 1), // #nosec
		Actor: account.Actor,
		Tpid: "123@test",
	})
	_, _, err = api.SignTransaction(NewTransaction([]*Action{overFlowFee}, opts), opts.ChainID, CompressionNone)
	if err == nil {
		t.Error("potential int overflow not blocked when signing transaction")
	}

	// negative int into int
	// most likely scenario ... could happen if using ABI to encode, but we'll just create a fake action to test
	type fakeAction struct {
		MaxFee int64 `json:"max_fee"`
	}
	underFlowAbi := NewAction("fake", "action", account.Actor, fakeAction{MaxFee: -1})
	_, _, err = api.SignTransaction(NewTransaction([]*Action{underFlowAbi}, opts), opts.ChainID, CompressionNone)
	if err == nil {
		t.Error("potential uint underflow not blocked when signing transaction")
	}

}

func Test_NewSubmitMultiplier(t *testing.T) {
	var multiplier float64
	acc, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}

	// grab current multiplier, don't want to guess...
	gtr, err := api.GetTableRows(eos.GetTableRowsRequest{
		Code:       "fio.fee",
		Scope:      "fio.fee",
		Table:      "feevoters",
		LowerBound: string(acc.Actor),
		UpperBound: string(acc.Actor),
		Limit:      1,
		KeyType:    "name",
		Index:      "1",
		JSON:       true,
	})
	if err != nil {
		t.Error(err)
		return
	}
	type FeeMultResp struct {
		FeeMultiplier string `json:"fee_multiplier"`
	}
	current := make([]FeeMultResp, 0)
	err = json.Unmarshal(gtr.Rows, &current)
	if err != nil {
		t.Error(err)
		return
	}
	if len(current) == 0 || current[0].FeeMultiplier == "0" {
		multiplier = 1
	} else {
		multiplier, err = strconv.ParseFloat(current[0].FeeMultiplier, 64)
		if err != nil {
			t.Error(err)
			return
		}
		multiplier += 0.1
	}
	_, err = api.SignPushActions(NewSetFeeMult(multiplier, acc.Actor))
	if err != nil {
		t.Error(err)
	}

}
