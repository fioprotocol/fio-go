package main

// example of transferring FIO tokens

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

		to = `FIO6G9pXXM92Gy5eMwNquGULoCj3ZStwPLPdEb9mVXyEHqWN7HSuA`
	)

	fatal := func(e error) {
		if e != nil {
			log.Fatal(e)
		}
	}

	account, api, _, err := fio.NewWifConnect(wif, url)
	fatal(err)

	// send ᵮ1.00
	resp, err := api.SignPushActions(fio.NewTransferTokensPubKey(account.Actor, to, fio.Tokens(1.0)))
	fatal(err)

	j, err := json.MarshalIndent(resp, "", "  ")
	fatal(err)
	fmt.Println(string(j))
}

