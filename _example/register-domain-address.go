package main

// example of registering a domain and an address

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fioprotocol/fio-go/v2"
	"log"
	"time"
)

func main() {

	const (
		url = `https://testnet.fioprotocol.io`
		wif = `5JP1fUXwPxuKuNryh5BEsFhZqnh59yVtpHqHxMMTmtjcni48bqC`
		domain = `example-domain`
		address = `name@example-domain`
	)

	// error helper
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	fatal := func(err error) {
		if err != nil {
			trace := log.Output(2, err.Error())
			log.Fatal(trace)
		}
	}
	// context helper
	cx := func() context.Context {
		ctx, _ := context.WithTimeout(context.Background(), 3 * time.Second)
		return ctx
	}

	// open a new connection to nodeos with credentials
	account, api, _, err := fio.NewWifConnect(cx(), wif, url)
	fatal(err)

	// register a new domain
	_, err = api.SignPushActions(cx(), fio.NewRegDomain(account.Actor, domain, account.PubKey))
	fatal(err)

	// register an address
	addr, ok := fio.NewRegAddress(account.Actor, address, account.PubKey)
	if !ok {
		log.Fatal("invalid address")
	}
	resp, err := api.SignPushActions(cx(), addr)
	fatal(err)

	j, err := json.MarshalIndent(resp, "", "")
	fatal(err)
	fmt.Println(string(j))

}
