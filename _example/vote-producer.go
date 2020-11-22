package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fioprotocol/fio-go/v2"
	"log"
	"time"
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

	resp, err := api.SignPushActions(cx(), fio.NewVoteProducer(producers, account.Actor, voter))
	fatal(err)

	j, err := json.MarshalIndent(resp, "", "  ")
	fatal(err)
	fmt.Println(string(j))
}

