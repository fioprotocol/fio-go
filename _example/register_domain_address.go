package main

// example of registering a domain and an address

import (
	"encoding/json"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"log"
)

func main() {

	const (
		url = `https://testnet.fioprotocol.io`
		wif = `5JP1fUXwPxuKuNryh5BEsFhZqnh59yVtpHqHxMMTmtjcni48bqC`
		domain = `example-domain`
		address = `name@example-domain`
	)

	fatal := func(e error) {
		if e != nil {
			log.Fatal(e)
		}
	}

	account, err := fio.NewAccountFromWif(wif)
	fatal(err)

	api, opts, err := fio.NewConnection(account.KeyBag, url)
	fatal(err)

	// register a new domain
	dom := fio.NewRegDomain(account.Actor, domain, account.PubKey)
	_, err = api.SignPushTransaction(
		fio.NewTransaction([]*fio.Action{dom}, opts),
		opts.ChainID,
		fio.CompressionNone,
	)
	fatal(err)

	// register an address
	addr := fio.MustNewRegAddress(account.Actor, address, account.PubKey)
	resp, err := api.SignPushTransaction(
		fio.NewTransaction([]*fio.Action{addr}, opts),
		opts.ChainID,
		fio.CompressionNone,
	)
	fatal(err)

	j, err := json.MarshalIndent(resp, "", "")
	fatal(err)
	fmt.Println(string(j))

}
