package main

// example of transferring FIO tokens

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

		to = `FIO6G9pXXM92Gy5eMwNquGULoCj3ZStwPLPdEb9mVXyEHqWN7HSuA`
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

	account, api, _, err := fio.NewWifConnect(cx(), wif, url)
	fatal(err)

	// send ᵮ1.00
	resp, err := api.SignPushActions(cx(), fio.NewTransferTokensPubKey(account.Actor, to, fio.Tokens(1.0)))
	fatal(err)

	j, err := json.MarshalIndent(resp, "", "  ")
	fatal(err)
	fmt.Println(string(j))
}

