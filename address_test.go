package fio

import (
	"encoding/json"
	"math/rand"
	"testing"
	"time"
)

func TestAddress_Valid(t *testing.T) {
	bad := []string{
		"has@two@ampersat",
		"no-@dashat",
		"no@-atdash",
		"-nodash@start",
		"nodash@end-",
		"no@dash--dash",
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
		"less<not@llowed",
		"great>not@llowed",
		"under_not@llowed",
		"missingdomain@",
		"@missingname",
		"@",
		"65656565656565656565656565656565@65656565656565656565656565656565",
	}
	for _, b := range bad {
		if Address(b).Valid() {
			t.Error(b + " should be an invalid address")
		}
	}
	good := []string{
		"a-b-c@xyz",
		"xyz@a-b-c",
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
			t.Error(g + " should be a valid address")
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

func word() string {
	rand.Seed(time.Now().UnixNano())
	var w string
	for i := 0; i < 8; i++ {
		w = w + string(byte(rand.Intn(26)+97))
	}
	return w
}

func TestAddress(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	account, api, opts, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}

	accountA, err := NewRandomAccount()
	if err != nil {
		t.Error(err)
		return
	}
	apiA, optsA, err := NewConnection(accountA.KeyBag, api.BaseURL)

	accountB, err := NewRandomAccount()
	if err != nil {
		t.Error(err)
		return
	}

	_, err = api.SignPushTransaction(NewTransaction(
		[]*Action{NewTransferTokensPubKey(
			account.Actor,
			accountA.PubKey,
			Tokens(
				GetMaxFee(FeeRegisterFioDomain)+
					GetMaxFee(FeeRenewFioDomain)+
					(3*GetMaxFee(FeeRegisterFioAddress))+
					GetMaxFee(FeeRenewFioAddress)+
					GetMaxFee(FeeTransferDom)+
					GetMaxFee(FeeTransferAddress)+
					GetMaxFee(FeeSetDomainPub)),
		)}, opts),
		opts.ChainID, CompressionNone,
	)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(time.Second)

	domain := word()
	names := []string{word(), word(), word()}

	// ensure available
	ok, err := api.AvailCheck(domain)
	if err != nil {
		t.Error("check available before register: " + err.Error())
	}
	if !ok {
		t.Error("domain was not available")
	}

	// register a domain
	_, err = apiA.SignPushTransaction(NewTransaction(
		[]*Action{NewRegDomain(accountA.Actor, domain, accountA.PubKey)}, optsA),
		optsA.ChainID, CompressionNone,
	)
	if err != nil {
		t.Error("Register domain: " + err.Error())
	}

	// ensure not available
	ok, err = api.AvailCheck(domain)
	if err != nil {
		t.Error("check available after register: " + err.Error())
	}
	if ok {
		t.Error("domain was still available")
	}

	// renew a domain
	_, err = apiA.SignPushTransaction(NewTransaction(
		[]*Action{NewRenewDomain(accountA.Actor, domain)}, optsA),
		optsA.ChainID, CompressionNone,
	)
	if err != nil {
		t.Error("Register domain: " + err.Error())
	}

	// confirm owner
	o, err := api.GetDomainOwner(domain)
	if err != nil {
		t.Error(err)
	} else if *o != accountA.Actor {
		t.Error("get owner had wrong result")
	}

	// set public
	_, err = apiA.SignPushTransaction(NewTransaction(
		[]*Action{NewSetDomainPub(accountA.Actor, domain, true)}, optsA),
		optsA.ChainID, CompressionNone,
	)
	if err != nil {
		t.Error("set public: " + err.Error())
	}

	// two addresses
	for _, n := range names[:2] {
		act, ok := NewRegAddress(accountA.Actor, Address(n+"@"+domain), accountA.PubKey)
		if !ok {
			t.Error("tried to register an invalid address")
			continue
		}
		_, err = apiA.SignPushTransaction(NewTransaction(
			[]*Action{act}, optsA),
			optsA.ChainID, CompressionNone,
		)
		if err != nil {
			t.Error("reg address: " + err.Error())
		}
	}

	// must reg (panic on fail)
	_, err = apiA.SignPushTransaction(NewTransaction(
		[]*Action{MustNewRegAddress(accountA.Actor, Address(names[2]+"@"+domain), accountA.PubKey)}, optsA),
		optsA.ChainID, CompressionNone,
	)
	if err != nil {
		t.Error("set public: " + err.Error())
	}

	// check we got bundled transactions
	rem, err := apiA.GetBundleRemaining(Address(names[2]+"@"+domain))
	if err != nil {
		t.Error("set get bundle: " + err.Error())
	}
	if rem == 0 {
		t.Error("expected remaining bundle to be a positive number")
	}

	// query by actor
	fioNames, ok, err := api.GetFioNamesForActor(string(accountA.Actor))
	if err != nil {
		t.Error(err)
	}
	if !ok {
		t.Error("did not get a result for get fio names for actor")
	} else {
		var found bool
		for _, a := range fioNames.FioAddresses {
			if names[2]+"@"+domain == a.FioAddress {
				found = true
			}
		}
		if !found {
			t.Error("could not lookup fio addresses for actor")
		}
	}

	// add one address
	naa, ok := NewAddAddress(accountA.Actor, Address(names[2]+"@"+domain), "token", "chain", "pubkey")
	if !ok {
		t.Error("invalid fio address while adding public address")
	}
	_, err = apiA.SignPushTransaction(NewTransaction(
		[]*Action{naa}, optsA),
		optsA.ChainID, CompressionNone,
	)
	if err != nil {
		t.Error("set public: " + err.Error())
	}

	// add three
	addresses := []TokenPubAddr{
		{ChainCode: "chain0", PublicAddress: "pubkey0", TokenCode: "token0"},
		{ChainCode: "chain1", PublicAddress: "pubkey1", TokenCode: "token1"},
		{ChainCode: "chain2", PublicAddress: "pubkey2", TokenCode: "token2"},
	}
	naas, ok := NewAddAddresses(accountA.Actor, Address(names[2]+"@"+domain), addresses)
	if !ok {
		t.Error("invalid fio address while adding public addresses")
	}
	_, err = apiA.SignPushTransaction(NewTransaction(
		[]*Action{naas}, optsA),
		optsA.ChainID, CompressionNone,
	)
	if err != nil {
		t.Error("set public: " + err.Error())
	}

	// lookup one of the addresses
	pubAddress, ok, err := api.PubAddressLookup(Address(names[2]+"@"+domain), "chain", "token")
	if err != nil {
		t.Error(err)
	}
	if !ok {
		t.Error("did not find address from pub address lookup")
	}
	if pubAddress.PublicAddress != "pubkey" {
		t.Error("got incorrect public address")
	}

	// get all of the addresses
	addrs, err := api.GetAllPublic(Address(names[2]+"@"+domain))
	if err != nil {
		t.Error("get all public:" + err.Error())
	}
	if len(addrs) != 5 {
		t.Errorf("get all public: only got %d of 5 addresses", len(addrs))
	}

	// renew it
	_, err = apiA.SignPushTransaction(NewTransaction(
		[]*Action{NewRenewAddress(accountA.Actor, names[2]+"@"+domain)}, optsA),
		optsA.ChainID, CompressionNone,
	)
	if err != nil {
		t.Error("set public: " + err.Error())
	}

	// transfer it
	_, err = apiA.SignPushTransaction(NewTransaction(
		[]*Action{NewTransferAddress(accountA.Actor, Address(names[2]+"@"+domain), accountB.PubKey)}, optsA),
		optsA.ChainID, CompressionNone,
	)
	if err != nil {
		t.Error("transfer address: " + err.Error())
	}
	time.Sleep(500 * time.Millisecond)

	// verify it transferred
	pubAddress, ok, err = api.PubAddressLookup(Address(names[2]+"@"+domain), "FIO", "FIO")
	if err != nil {
		t.Error(err)
	}
	if !ok {
		t.Error("did not find address from pub address lookup")
	}
	if pubAddress.PublicAddress != accountB.PubKey {
		t.Error("got incorrect public address after transfer")
	}

	// transfer the domain
	_, err = apiA.SignPushTransaction(NewTransaction(
		[]*Action{NewTransferDom(accountA.Actor, domain, accountB.PubKey)}, optsA),
		optsA.ChainID, CompressionNone,
	)
	if err != nil {
		t.Error("transfer domain: " + err.Error())
	}

	// verify
	newOwner, err := api.GetDomainOwner(domain)
	if err != nil {
		t.Error(err)
	}
	if accountB.Actor != *newOwner {
		t.Error("domain transfer failed")
	}

}

func TestRemoveAddr(t *testing.T) {
	account, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	i, _, err := account.GetNames(api)
	if err != nil {
		t.Error(err)
		return
	}
	if i == 0 {
		t.Error("cannot add addresses, account does not have any fio names")
	}

	junk := word()

	// need something there to remove
	naa, ok := NewAddAddress(account.Actor, Address(account.Addresses[0].FioAddress), junk, junk, junk)
	if !ok {
		t.Error("bad address")
		return
	}
	_, err = api.SignPushActions(naa)
	if err != nil {
		t.Error(err)
		return
	}

	// ensure it was added
	if a, _, err := api.PubAddressLookup(Address(account.Addresses[0].FioAddress), junk, junk); err != nil || a.PublicAddress == "" {
		t.Error("could not confirm address was added")
	}

	// remove the address
	rm, err := NewRemoveAddrReq(Address(account.Addresses[0].FioAddress), []TokenPubAddr{{PublicAddress: junk, TokenCode: junk, ChainCode: junk}}, account.Actor)
	if err != nil {
		t.Error(err)
	}
	_, err = api.SignPushActions(rm)
	if err != nil {
		t.Error(err)
		return
	}

	// ensure it's gone
	if _, _, err := api.PubAddressLookup(Address(account.Addresses[0].FioAddress), junk, junk); err != nil {
		t.Error("could not confirm address was removed")
	}

	// add more junk
	allTheJunk := make([]string, 0)
	for i := 0; i < 5; i++ {
		junk = word()
		allTheJunk = append(allTheJunk, junk)
		// need something there to remove
		naa, ok := NewAddAddress(account.Actor, Address(account.Addresses[0].FioAddress), junk, junk, junk)
		if !ok {
			t.Error("bad address")
			return
		}
		_, err = api.SignPushActions(naa)
		if err != nil {
			t.Error(err)
			return
		}
		time.Sleep(500 * time.Millisecond)
	}

	// make sure it is there
	for _, junk = range allTheJunk {
		if a, _, err := api.PubAddressLookup(Address(account.Addresses[0].FioAddress), junk, junk); err != nil || a.PublicAddress == "" {
			t.Error("could not confirm address was added ", junk)
		}
	}

	// one call to remove them all
	rmfr, err := NewRemoveAllAddrReq(Address(account.Addresses[0].FioAddress), account.Actor)
	if err != nil {
		t.Error(err)
	}
	_, err = api.SignPushActions(rmfr)
	if err != nil {
		t.Error(err)
		return
	}
	for _, junk = range allTheJunk {
		if _, _, err := api.PubAddressLookup(Address(account.Addresses[0].FioAddress), junk, junk); err != nil {
			t.Error("could not confirm address was removed ", junk)
		}
	}

}
