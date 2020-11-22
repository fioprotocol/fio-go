package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fioprotocol/fio-go/v2"
	"log"
	"time"
)

// example of retrieving and decrypting a funds request

func main() {
	const (
		url = `https://testnet.fioprotocol.io`
		wif = `5JP1fUXwPxuKuNryh5BEsFhZqnh59yVtpHqHxMMTmtjcni48bqC`
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

	// get the first pending funds request and print
	r, hasPending, err := api.GetPendingFioRequests(cx(), account.PubKey, 1, 1)
	fatal(err)
	if !hasPending {
		log.Fatal("no pending requests found")
	}

	fmt.Println("Request:")
	j, err := json.MarshalIndent(r, "", "  ")
	fatal(err)
	fmt.Println(string(j))

	// decrypt and print the content
	obtReq, err := fio.DecryptContent(account, r.Requests[0].PayeeFioPublicKey, r.Requests[0].Content, fio.ObtRequestType)
	fatal(err)

	fmt.Println("Decrypted Content:")
	j, err = json.MarshalIndent(obtReq, "", "  ")
	fatal(err)
	fmt.Println(string(j))
}

