package fio

import (
	"encoding/json"
	"github.com/eoscanada/eos-go"
	"testing"
)

func TestAPI_GetFioAccount(t *testing.T) {
	api, _, _ := NewConnection(nil, "https://testnet.fio.dev")
	a, err := api.GetFioAccount("gik4jgcjciwb")
	if err != nil {
		t.Error(err)
	}
	if a == nil {
		t.Error("nil response")
		return
	}
	_, err = json.MarshalIndent(a, "", "  ")
	if err != nil {
		t.Error(err)
	}
	if a.AccountName != eos.AccountName("gik4jgcjciwb") {
		t.Error("account name was not correct")
	}
}
