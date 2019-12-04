# FIO-GO

Library for interacting with the FIO network using the go language.

## Example

This demonstrates using the library to send FIO tokens from one account to another:

```go
package main

import (
	"encoding/json"
	"fmt"
	"github.com/dapixio/fio-go"
	"github.com/eoscanada/eos-go"
	"log"
)

func main() {

	const (
		url       = `https://testnet.fioprotocol.io`
		recipient = `bp:dapixbp`
		senderWif = `5KSQbcNjunVU38b2RdADLqZvz893ZgjdTAoSrV51mne4T97i1qC`
	)

	// Create a FIO Account type from a WIF:
	sender, err := fio.NewAccountFromWif(senderWif)
	if err != nil {
		log.Fatal(err)
	}

	// Setup a connection to the nodeos API
	api, options, err := fio.NewConnection(sender.KeyBag, url)
	if err != nil {
		log.Fatal(err)
	}

	// Get the FIO public key for the Address
	pubKey, found, err := api.PubAddressLookup(recipient, "FIO")
	if err != nil {
		log.Fatal(err)
	} else if !found {
		log.Fatal("Couldn't find an account for " + recipient)
	}

	// Build the action, embed it in a TX, pack, and sign
	_, packedTx, err := api.SignTransaction(
		fio.NewTransaction(
			[]*fio.Action{
				fio.NewTransferTokensPubKey(
					sender.Actor,
					pubKey.PublicAddress,
					fio.Tokens(0.5), // send 1/2 FIO
				),
			},
			options,
		),
		options.ChainID,
		eos.CompressionNone,
	)
	if err != nil {
		log.Fatal(err)
	}

	// Send to the network
	response, err := api.PushTransaction(packedTx)
	if err != nil {
		log.Fatal(err)
	}

	// Output the result
	j, _ := json.MarshalIndent(response, "", "  ")
	fmt.Println(string(j))
}
```
