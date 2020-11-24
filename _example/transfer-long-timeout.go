package main

// example of transferring FIO tokens, AND overriding the default timeout from 30 seconds to 1 hour.

import (
	"encoding/json"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"log"
	"time"
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

	// Note that txOpts are needed.
	account, api, txOpts, err := fio.NewWifConnect(wif, url)
	fatal(err)

	// The timeout is part of the transaction, so instead of using SignPushActions, create an un-packed transaction
	tx := fio.NewTransaction(
		[]*fio.Action{
			fio.NewTransferTokensPubKey(account.Actor, to, fio.Tokens(1.0)),
		},
		txOpts,
	)

	// override the expiration with the longest allowed timeout, FC will reject anything more than 3600 seconds
	tx.SetExpiration(time.Hour)

	// send ᵮ1.00
	// Now instead of SignPushActions, it is sent with SignPushTransaction
	resp, err := api.SignPushTransaction(tx, txOpts.ChainID, fio.CompressionNone)
	fatal(err)

	j, err := json.MarshalIndent(resp, "", "  ")
	fatal(err)
	fmt.Println(string(j))
}
