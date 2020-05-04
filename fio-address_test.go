package fio

import (
	"encoding/json"
	"testing"
)

func TestAddress_Valid(t *testing.T) {
	bad := []string{
		"has@two@ampersat",
		"no-@dashat",
		"no@-atdash",
		"-nodash@start",
		"nodash@end-",
		"bang!not@allowed",
		"hash#not@llowed",
		"dollar$not@llowed",
		"perc%not@llowed",
		"caret^not@llowed",
		"amp&not@llowed",
		"splat*not@llowed",
		"open(not@llowed",
		"close)not@llowed",
		"under_not@llowed",
		"open[not@llowed",
		"close]not@llowed",
		"open{not@llowed",
		"close}not@llowed",
		"slash/not@llowed",
		"q?not@llowed",
		"dot.not@llowed",
		"less>not@llowed",
		"great>not@llowed",
		"under_not@llowed",
		"missingdomain@",
		"@missingname",
		"@",
		"65656565656565656565656565656565@65656565656565656565656565656565",
	}
	for _, b := range bad {
		if Address(b).Valid() {
			t.Error(b+" should be an invalid address")
		}
	}
	good := []string{
		"a@b",
		"a-b@c",
		"a@b-c",
		"a-b@c-d",
		"1@2",
		"1-2@3",
		"1@2-3",
		"1-2@3-4",
		"a@bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb@c",
	}
	for _, g := range good {
		if !Address(g).Valid() {
			t.Error(g+" should be a valid address")
		}
	}
}

func TestAPI_GetFioNames(t *testing.T) {

	// these are devnet accounts that have been added to testnet so tests can run vs either:
	var (
		pubkey  = `FIO5oBUYbtGTxMS66pPkjC2p8pbA3zCtc8XD4dq9fMut867GRdh82`
		domain  = `dapixdev`
		address = `ada@dapixdev`
	)

	_, api, _, err := newApi()
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
			hasDomain = false
			hasAddress = true
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
