package main

// example of sending a funds request

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

		payee    = `vendor@domain` // payee is account that will get paid
		payeeEth = `0xa4009bc130e8b900715c2D48d17c02f1c3B138c7`
		amount   = "1.0" // amount is full tokens/coins as a string
		memo     = `payment for cart: 123456`
		payer    = `buyer@domain` // payer is account that receives request
	)

	fatal := func(e error) {
		if e != nil {
			log.Fatal(e)
		}
	}

	account, err := fio.NewAccountFromWif(wif)
	fatal(err)

	api, opts, err := fio.NewConnection(nil, url)
	fatal(err)

	// get the FIO public key for the payer, this is used to encrypt the request:
	payerPub, ok, err := api.PubAddressLookup(payer, "FIO", "FIO")
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
	fundsRequest := fio.NewFundsReq(account.Actor, payer, payee, encrypted)
	resp, err := api.SignPushTransaction(
		fio.NewTransaction([]*fio.Action{fundsRequest}, opts),
		opts.ChainID,
		fio.CompressionNone,
	)
	fatal(err)

	// print the result
	j, err := json.MarshalIndent(resp, "", "  ")
	fatal(err)
	fmt.Println(string(j))
}
