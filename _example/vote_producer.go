package main

import (
	"encoding/json"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"log"
)

// example for voting for producers

func main() {
	const (
		url = `https://testnet.fioprotocol.io`
		wif = `5JP1fUXwPxuKuNryh5BEsFhZqnh59yVtpHqHxMMTmtjcni48bqC`
		voter = `me@domain`
	)
	producers := []string{
		"bp1@domain",
		"bp2@domain",
		"bp3@domain",
	}

	fatal := func(e error) {
		if e != nil {
			log.Fatal(e)
		}
	}

	account, api, _, err := fio.NewWifConnect(wif, url)
	fatal(err)

	resp, err := api.SignPushActions(fio.NewVoteProducer(producers, account.Actor, voter))
	fatal(err)

	j, err := json.MarshalIndent(resp, "", "  ")
	fatal(err)
	fmt.Println(string(j))
}

