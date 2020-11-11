package fio

import (
	"fmt"
	"github.com/fioprotocol/fio-go/eos"
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
	_, packed, err := api.SignTransaction(NewTransaction(
		[]*Action{NewSetFeeVote([]*FeeValue{
			{
				EndPoint: "register_fio_domain",
				Value:    40000000000,
			},
		},acc.Actor)}, opts),
		opts.ChainID, CompressionNone,
	)
	if err != nil {
		t.Error(err)
		return
	}
	j, err := api.PushTransactionRaw(packed)
	if err != nil {
		t.Error(err)
		fmt.Println(string(j))
	}

	opts.Compress = CompressionZlib
	resp, err := api.SignPushActionsWithOpts([]*eos.Action{
		NewSetFeeVote([]*FeeValue{
			{
				EndPoint: "register_fio_domain",
				Value:    40000000000,
			},
		},acc.Actor).ToEos(),
	}, &opts.TxOptions)
	if err != nil {
		t.Error(err)
		fmt.Println(resp)
	}

}