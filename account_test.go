package fio

import (
	"encoding/json"
	"fmt"
	"github.com/eoscanada/eos-go"
	"github.com/fioprotocol/fio-go/eos-go/ecc"
	"os"
	"testing"
)

func newApi() (*Account, *API, *TxOptions, error) {
	nodeos := "http://dev:8889"
	if os.Getenv("NODEOS") != "" {
		nodeos = os.Getenv("NODEOS")
	}
	account, err := NewAccountFromWif("5JBbUG5SDpLWxvBKihMeXLENinUzdNKNeozLas23Mj6ZNhz3hLS")
	if err != nil {
		return nil, nil, nil, err
	}
	api, opts, err := NewConnection(account.KeyBag, nodeos)
	if err != nil {
		return nil, nil, nil, err
	}
	return account, api, opts, err
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
	if a.AccountName != eos.AccountName("qbxn5zhw2ypw") {
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

func TestApi_getMaxActions(t *testing.T) {
	_, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	if !api.HasHistory() {
		fmt.Println("history api not available, skipping getMaxActions test")
		return
	}
	h, err := api.getMaxActions("eosio")
	if err != nil {
		t.Error(err)
		return
	}
	if h < 1000 {
		t.Errorf("eosio did not have enough action traces expected > 1000, got %d", h)
	}
}

func TestAPI_VerifyAddressHistory(t *testing.T) {
	_, api, opts, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	if !api.HasHistory() || opts.ChainID.String() != ChainIdTestnet {
		fmt.Println("skipping test for VerifyAddressHistory, this requires a history node running on testnet")
		return
	}
	pub, err := ecc.NewPublicKey("FIO5oBUYbtGTxMS66pPkjC2p8pbA3zCtc8XD4dq9fMut867GRdh82")
	if err != nil {
		t.Error(err)
		return
	}

	status, err := api.VerifyAddressHistory(VerifyAddressHistoryRequest{
		Pubkey:     pub,
		FioAddress: `ada@dapixdev`,
		Token:      "FIO",
		Chain:      "FIO",
		PubAddress: "FIO5oBUYbtGTxMS66pPkjC2p8pbA3zCtc8XD4dq9fMut867GRdh82",
		ChainId:    ChainIdTestnet,
		Limit:      -1,
	})
	if err != nil {
		t.Error(err)
		return
	}

	if status != AddressHistoryCurrent {
		t.Error("ada@dapix dev should be valid for r41zuwovtn44")
	}
}