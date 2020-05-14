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

	api, _, err := fio.NewConnection(account.KeyBag, url)
	fatal(err)

	// register a new domain
	_, err = api.SignPushActions(fio.NewRegDomain(account.Actor, domain, account.PubKey))
	fatal(err)

	// register an address
	addr, ok := fio.NewRegAddress(account.Actor, address, account.PubKey)
	if !ok {
		log.Fatal("invalid address")
	}
	resp, err := api.SignPushActions(addr)
	fatal(err)

	j, err := json.MarshalIndent(resp, "", "")
	fatal(err)
	fmt.Println(string(j))

}
