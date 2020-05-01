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

func TestNewAccountFromWif(t *testing.T) {
	account, err := NewAccountFromWif(`5JfNfukKhyCe4MSTBMiMdT77d8MCetEpceDQqRh4DuJQ1CAEdQF`)
	if err != nil {
		t.Error(err)
		return
	}
	if account.Actor != eos.AccountName("tccyed5wnyj5") {
		t.Error("bad actor")
	}
	if account.PubKey != `FIO6JN7BrPKPM8BqPs9zSPwbK3nWJ4EKvpjb4k9CFBQ6BbtrL2AHV` {
		t.Error("bad pub key")
	}
}

// test +dev
func TestAccount_GetNames(t *testing.T) {
	api, _, err := NewConnection(nil, "http://dev:8889")
	if err != nil {
		t.Error(err)
		return
	}
	account, _ := NewAccountFromWif(`5KQ6f9ZgUtagD3LZ4wcMKhhvK9qy4BuwL3L1pkm6E2v62HCne2R`)
	names, _, err := account.GetNames(api)
	if err != nil {
		t.Error(err)
		return
	}
	if names == 0 {
		t.Error("did not find name")
		return
	}
	if account.Addresses[0].FioAddress != "bp1@dapixdev" {
		t.Error("did not have correct address")
	}
}

