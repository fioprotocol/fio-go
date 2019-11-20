# FIO-GO

Library for interacting with the FIO network using the go language.

## Example

This demonstrates using the library to send FIO tokens from one account to another:

```go
package main

import (
	"fmt"
	"github.com/dapixio/fio-go"
	"github.com/eoscanada/eos-go"
	"log"
)

func main() {

	url := `https://testnet.fioprotocol.io`
	recipient := `FIO5NMm9Vf3NjYFnhoc7yxTCrLW963KPUCzeMGv3SJ6zR3GMez4ub`
	senderWif := `5KSQbcNjunVU38b2RdADLqZvz893ZgjdTAoSrV51mne4T97i1qC`

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

	// Create the transaction, in this case transfer 1 FIO token
	action := fio.NewTransferTokensPubKey(sender.Actor, recipient, fio.ConvertAmount(1.0))
	tx := eos.NewTransaction([]*eos.Action{action}, options)

	// Pack and sign
	_, packedTx, err := api.SignTransaction(tx, options.ChainID, eos.CompressionNone)
	if err != nil {
		log.Fatal(err)
	}

	// Send to the network
	response, err := api.PushTransaction(packedTx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", response)
}
```
