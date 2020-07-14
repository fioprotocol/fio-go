package fio

import (
	"encoding/json"
	feos "github.com/fioprotocol/fio-go/imports/eos-fio"
	"os"
	"testing"
)

func newApi() (*Account, *API, *TxOptions, error) {
	nodeos := "http://dev:8889"
	if os.Getenv("NODEOS") != "" {
		nodeos = os.Getenv("NODEOS")
	}
	return NewWifConnect("5JBbUG5SDpLWxvBKihMeXLENinUzdNKNeozLas23Mj6ZNhz3hLS", nodeos)
}

func TestAPI_GetFioAccount(t *testing.T) {
	_, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	a, err := api.GetFioAccount("qbxn5zhw2ypw")
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
	if a.AccountName != feos.AccountName("qbxn5zhw2ypw") {
		t.Error("account name was not correct")
	}
}

func TestNewAccountFromWif(t *testing.T) {
	account, err := NewAccountFromWif(`5JfNfukKhyCe4MSTBMiMdT77d8MCetEpceDQqRh4DuJQ1CAEdQF`)
	if err != nil {
		t.Error(err)
		return
	}
	if account.Actor != feos.AccountName("tccyed5wnyj5") {
		t.Error("bad actor")
	}
	if account.PubKey != `FIO6JN7BrPKPM8BqPs9zSPwbK3nWJ4EKvpjb4k9CFBQ6BbtrL2AHV` {
		t.Error("bad pub key")
	}
}

func TestAccount_GetNames(t *testing.T) {
	_, api, _, err := newApi()
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

func TestActorFromPub(t *testing.T) {
	type testAccounts struct {
		Pubkey string
		Account string
		Valid bool
	}
	tests := []testAccounts{
		{"FIO586ZYe3CA2D3cpuYJk565Ny7RhgWxCwnX7kojZSaun2RbTocAf", "y5x3sk44d43p", true},
		{"EOS6sUfyCJZHj4xQWiK79Zmz9CsfFfQ9ci2jZqGiLo3yYiw9pcgAG", "", false},
		{"FIO586ZYe3CA2D3cpuYJk565Ny7RhgWxCwnX7kojZSaun2RbTocA1", "", false},
		{"FIOhellothere", "", false},
		{"PUB_K1_6MRyAjQq8ud7hVNYcfnVPJqcVpscN5So8BhtHuGYqET5GDW5CV", "ymwzn5vje5ay", true},
		{"PUB_K1_6MRyAjQq8ud7hVNYcfnVPJqcVpscN5So8BhtHuGYqET5GDW5C1", "", false},
	}
	for _, pk := range tests {
		act, err := ActorFromPub(pk.Pubkey)
		switch pk.Valid {
		case true:
			if err != nil {
				t.Error(pk.Pubkey, "should be a valid public key")
			}
			if pk.Account != string(act) {
				t.Error(pk.Pubkey, "should map to", pk.Account, "but got", act)
			}
		case false:
			if err == nil {
				t.Error(pk.Pubkey, "should be an invalid public key")
			}
			if act != "" {
				t.Error(pk.Pubkey, "should not have mapped to an account, got", act)
			}
		}
	}
}