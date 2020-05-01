package fio

import (
	"encoding/json"
	"os"
	"testing"
)

func TestAPI_GetFioNames(t *testing.T) {

	// these are devnet accounts that have been added to testnet so tests can run vs either:
	var (
		pubkey  = `FIO5oBUYbtGTxMS66pPkjC2p8pbA3zCtc8XD4dq9fMut867GRdh82`
		domain  = `dapixdev`
		address = `ada@dapixdev`
	)
	nodeos := "https://testnet.fio.dev"
	if os.Getenv("NODEOS") != "" {
		nodeos = os.Getenv("NODEOS")
	}

	api, _, err := NewConnection(nil, nodeos)
	if err != nil {
		t.Error("cannot connect to run FIO names test: " + err.Error())
		return
	}

	for _, toTest := range []string{"GetFioNames", "GetFioDomains", "GetFioAddresses"} {
		var hasDomain bool
		var hasAddress bool
		names := &FioNames{}
		switch toTest {
		case "GetFioNames":
			var ok bool
			var n FioNames
			n, ok, err = api.GetFioNames(pubkey)
			if err != nil {
				t.Error(toTest + " " + err.Error())
				break
			}
			if !ok {
				t.Error("GetFioNames: no results")
				break
			}
			names = &n
			hasDomain = true
			hasAddress = true
		case "GetFioDomains":
			names, err = api.GetFioDomains(pubkey, 0, 100)
			if err != nil {
				t.Error(toTest + " " + err.Error())
				break
			}
			hasDomain = true
			hasAddress = false
		case "GetFioAddresses":
			names, err = api.GetFioAddresses(pubkey, 0, 100)
			if err != nil {
				t.Error(toTest + " " + err.Error())
				break
			}
		}
		if hasDomain && (names == nil || names.FioDomains == nil || len(names.FioDomains) == 0) {
			t.Error("GetFioNames did not find at least one domain" + printResult(toTest, names))
			break
		}
		if hasAddress && (names == nil || names.FioAddresses == nil || len(names.FioAddresses) == 0) {
			t.Error("GetFioNames did not find at least one address" + printResult(toTest, names))
			break
		}
		found := make(map[string]bool)
		for _, d := range names.FioDomains {
			found[d.FioDomain] = true
		}
		for _, d := range names.FioAddresses {
			found[d.FioAddress] = true
		}
		if hasDomain && !found[domain] {
			t.Error(toTest + " did not find expected domain: " + domain + printResult(toTest, names))
		}
		if hasAddress && !found[address] {
			t.Error(toTest + " did not find expected address: " + address + printResult(toTest, names))
		}
	}
}

func printResult(from string, result *FioNames) string {
	if result == nil {
		return "\n" + from + " returned a nil response"
	}
	j, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "\n" + from + " could not unmarshal " + err.Error()
	}
	return "\n" + from + "\n" + string(j)
}
