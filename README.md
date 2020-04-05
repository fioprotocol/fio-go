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
	"log"
)

func main() {
	const (
		url = `https://testnet.fioprotocol.io`
		wif = `5JP1fUXwPxuKuNryh5BEsFhZqnh59yVtpHqHxMMTmtjcni48bqC`

		to = `FIO6G9pXXM92Gy5eMwNquGULoCj3ZStwPLPdEb9mVXyEHqWN7HSuA`
	)

	fatal := func(e error) {
		if e != nil {
			log.Fatal(e)
		}
	}

	// import the private key
	account, err := fio.NewAccountFromWif(wif)
	fatal(err)

	// connect to the network
	api, opts, err := fio.NewConnection(account.KeyBag, url)
	fatal(err)

	// send ᵮ1.00
	xfer := fio.NewTransferTokensPubKey(account.Actor, to, fio.Tokens(1.0))
	resp, err := api.SignPushTransaction(
		fio.NewTransaction([]*fio.Action{xfer}, opts),
		opts.ChainID,
		fio.CompressionNone,
	)
	fatal(err)

	// print the result
	j, err := json.MarshalIndent(resp, "", "  ")
	fatal(err)
	fmt.Println(string(j))
}
```
