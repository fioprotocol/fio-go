package fio

import (
	"fmt"
	"log"
)

// this example demonstrates making a connection to nodeos with an account.
func ExampleNewConnection() {
	// ********************
	// Setup the connection
	// ********************

	const (
		myPrivateKey = "5K1kaTgtd7NY7RMwSvbfs3SKEaeedBm1D9S3yp8Rh6Etb3Bwd3i"
		url          = "https://testnet.fioprotocol.io"
	)

	// Use a WIF to import an account
	owner, err := NewAccountFromWif(myPrivateKey)
	if err != nil {
		log.Fatal(err)
	}

	// Setup a connection that will use the account's credentials
	api, opts, err := NewConnection(owner.KeyBag, url)
	if err != nil {
		log.Fatal(err)
	}

	info, _ := api.GetInfo()
	fmt.Println("Current Producer", info.HeadBlockProducer)
	fmt.Println("Chain ID", opts.ChainID)

}

// This example registers a new domain
func ExampleNewRegDomain() {

	const (
		fioDomain    = "mydomain"
		myPrivateKey = "5K1kaTgtd7NY7RMwSvbfs3SKEaeedBm1D9S3yp8Rh6Etb3Bwd3i"
		url          = "https://testnet.fioprotocol.io"
	)
	owner, err := NewAccountFromWif(myPrivateKey)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connecting to nodeos endpoint at:", url, "as", owner.Actor)
	api, txOpts, err := NewConnection(owner.KeyBag, url)

	// Get our current block for searching transaction status
	currentBlock := api.GetCurrentBlock()
	if currentBlock == 0 {
		log.Fatal("First block listed as 0, something's not right!")
	}

	// ******************
	// Register a domain:
	// ******************

	// - Create the EOS Action that will be sent:
	action := NewRegDomain(owner.Actor, fioDomain, owner.PubKey)

	// - Embed the action in a transaction:
	tx := NewTransaction([]*Action{action}, txOpts)

	// - Pack and sign the tx:
	_, packedTx, err := api.SignTransaction(tx, txOpts.ChainID, CompressionNone)

	// - Broadcast the tx to the network:
	result, err := api.PushTransaction(packedTx)
	if err != nil {
		log.Fatal("push new domain: " + err.Error())
	}

	// - wait for the transaction to be published:
	block, err := api.WaitForConfirm(currentBlock, result.TransactionID)
	if err != nil {
		log.Fatal("waiting for domain: " + err.Error())
	}
	fmt.Println("Found transaction in block:", block)
	// Output: Found transaction in block: 4503046

}

/*
// this example looks up the public key for a FIO address, and checks if it matches our private key.
func ExampleAPI_PubAddressLookup() {

	const (
		myPrivateKey = "5K1kaTgtd7NY7RMwSvbfs3SKEaeedBm1D9S3yp8Rh6Etb3Bwd3i"
		url          = "https://testnet.fioprotocol.io"
		address      = Address("myaddress:mydomain")
	)

	owner, err := NewAccountFromWif(myPrivateKey)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connecting to nodeos endpoint at:", url, "as", owner.Actor)
	api, _, err := NewConnection(owner.KeyBag, url)
	if err != nil {
		log.Fatal(err)
	}
	// **************************
	// Get info about the address
	// **************************

	// - Lookup the key, and see if it matches:
	pub, _, err := api.PubAddressLookup(address, "FIO")
	if err != nil {
		log.Fatal("lookup: " + err.Error())
	}
	if owner.PubKey == pub.PublicAddress {
		log.Println("Address was successfully registered, and is owned by:", pub.PublicAddress)
	}
	// Output: 2019/12/02 16:09:45 Address was successfully registered, and is owned by: FIO5.....
}
*/

// This example registers a FIO address on a domain
func ExampleNewRegAddress() {

	// *********************************
	// Register an address on the domain
	// *********************************

	const (
		address      = Address("myaddress:mydomain")
		myPrivateKey = "5K1kaTgtd7NY7RMwSvbfs3SKEaeedBm1D9S3yp8Rh6Etb3Bwd3i"
		url          = "https://testnet.fioprotocol.io"
	)

	owner, err := NewAccountFromWif(myPrivateKey)
	if err != nil {
		log.Fatal(err)
	}
	api, txOpts, err := NewConnection(owner.KeyBag, url)
	if err != nil {
		log.Fatal(err)
	}

	// - Create the action, ensure what was provided was a valid address
	action, ok := NewRegAddress(owner.Actor, address, owner.PubKey)
	if !ok {
		fmt.Printf("%s is not a valid FIO address!", address)
	}

	// - embed, pack, and transmit
	_, packedTx, err := api.SignTransaction(
		NewTransaction([]*Action{action}, txOpts),
		txOpts.ChainID,
		CompressionNone,
	)
	result, err := api.PushTransaction(packedTx)
	if err != nil {
		log.Fatal("push new address: " + err.Error())
	}

	// - wait for the transaction to be published:
	block, err := api.WaitForConfirm(api.GetCurrentBlock()-2, result.TransactionID)
	if err != nil {
		log.Fatal("waiting for address: " + err.Error())
	}
	fmt.Println("Found transaction in block:", block)
	// Output: Found transaction in block: 4503053
}
