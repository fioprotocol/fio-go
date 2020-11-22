package main

// example of sending a funds request

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

		payee    = `vendor@domain` // payee is account that will get paid
		payeeEth = `0xa4009bc130e8b900715c2D48d17c02f1c3B138c7`
		amount   = "1.0" // amount is full tokens/coins as a string
		memo     = `payment for cart: 123456`
		payer    = `buyer@domain` // payer is account that receives request
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

	// setup a connection associated with our private key
	account, api, _, err := fio.NewWifConnect(cx(), wif, url)

	// get the FIO public key for the payer, this is used to encrypt the request:
	payerPub, ok, err := api.PubAddressLookup(cx(), payer, "FIO", "FIO")
	fatal(err)
	if !ok {
		log.Fatalf("Couldn't find PubKey for %s.\n", payer)
	}
	fmt.Printf("%s has FIO public key %s\n", payer, payerPub.PublicAddress)

	// encrypt the "content" field using ECEIS:
	// returns base64 encoded string
	encrypted, err := fio.ObtRequestContent{
		PayeePublicAddress: payeeEth,
		Amount:             amount,
		ChainCode:          "ETH",  // what chain, ie - ETH, EOS, BTC
		TokenCode:          "USDT", // token, if requesting the native coin will match ChainCode
		Memo:               memo,
	}.Encrypt(account, payerPub.PublicAddress)
	fatal(err)

	// send the request:
	resp, err := api.SignPushActions(cx(), fio.NewFundsReq(account.Actor, payer, payee, encrypted))
	fatal(err)

	// print the result
	j, err := json.MarshalIndent(resp, "", "  ")
	fatal(err)
	fmt.Println(string(j))
}

