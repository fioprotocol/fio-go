package main

import (
	"encoding/json"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"log"
)

// example of retrieving and decrypting a funds request

func main() {
	const (
		url = `https://testnet.fioprotocol.io`
		wif = `5JP1fUXwPxuKuNryh5BEsFhZqnh59yVtpHqHxMMTmtjcni48bqC`
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

	// get the first pending funds request and print
	r, hasPending, err := api.GetPendingFioRequests(account.PubKey, 1, 1)
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

