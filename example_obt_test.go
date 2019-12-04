package fio

import (
	"log"
)

func ExampleNewFundsReq() {

	// payer is who will receive the request to send funds
	const payer = "bp:dapixbp"

	// payee is who will receive funds, need a private key to perform encryption of the request
	payee, _ := NewAccountFromWif("5K1kaTgtd7NY7RMwSvbfs3SKEaeedBm1D9S3yp8Rh6Etb3Bwd3i")

	api, txOpts, err := NewConnection(payee.KeyBag, "https://testnet.fioprotocol.io")
	if err != nil {
		log.Fatal("connect: " + err.Error())
	}

	// Refresh the Payee's registered FIO addresses ... this populates the fio.Account.[]Addresses slice
	if addressCount, _, err := payee.GetNames(api); addressCount == 0 || err != nil {
		log.Fatal("Couldn't get a FIO address for the payee, a FIO address is required to request funds.")
	}

	// Get a FIO address for the Payer ...
	payerPub, ok, _ := api.PubAddressLookup(payer, "FIO")
	if !ok {
		log.Fatal("Couldn't find PubKey for payer.")
	}

	// send a request for a bitcoin
	// TODO: are bitcoin requests in Sat, or as a float?
	// ObtContent.Encrypt uses an ECIES derived shared-key to AES256 encrypt, then signs with a sha256 HMAC
	content, err := ObtContent{
		PayeePublicAddress: "1HPiiTTYioVBuDU29U7iQqk7tsoEaWoKQs",
		Amount:             "1000000",
		TokenCode:          "BTC",
		Memo:               "invoice: 123",
	}.Encrypt(payee, payerPub.PublicAddress)
	if err != nil {
		log.Fatal("encrypt: " + err.Error())
	}

	// create the transaction, containing the NewFundsReq action:
	tx := NewTransaction(
		[]*Action{NewFundsReq(payee.Actor, payer, payee.Addresses[0].FioAddress, content)},
		txOpts,
	)
	_, packedTx, err := api.SignTransaction(tx, txOpts.ChainID, CompressionNone)
	if err != nil {
		log.Fatal("sign: " + err.Error())
	}

	// broadcast to network
	result, err := api.PushTransaction(packedTx)
	if err != nil {
		log.Fatal("push: " + err.Error())
	}

	// wait for the transaction to be confirmed in at least one block:
	_, err = api.WaitForConfirm(api.GetCurrentBlock()-2, result.TransactionID)
	if err != nil {
		log.Fatal("wait: " + err.Error())
	}

}
