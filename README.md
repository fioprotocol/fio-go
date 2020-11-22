# FIO-GO
![Gosec](https://github.com/fioprotocol/fio-go/workflows/Gosec/badge.svg)

Library for interacting with the FIO network using the go language.

## Breaking Changes

The v2 release incorporates the latest eos-go library, and has removed the forked copy. This directory has v1 to
ensure long-term compatibility, but it is highly recommended to update downstream programs to use v2.

One major change in v2 is that most API calls require a `context.Context` to be supplied. And eos-go has added features
allowing overrides of the compatibility prefix in the `ecc` package, eliminating the need for a fork of eos-go to
remain in the repository.

## Example

This demonstrates using the library to send FIO tokens from one account to another:

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fioprotocol/fio-go/v2"
	"log"
)

func main() {
	const (
		url = `https://testnet.fioprotocol.io`
		wif = `5JP1fUXwPxuKuNryh5BEsFhZqnh59yVtpHqHxMMTmtjcni48bqC`
		to  = `FIO6G9pXXM92Gy5eMwNquGULoCj3ZStwPLPdEb9mVXyEHqWN7HSuA`
	)

	fatal := func(e error) {
		if e != nil {
			log.Fatal(e)
		}
	}

	// connect to the network, using credentials
	account, api, _, err := fio.NewWifConnect(context.Background(), wif, url)
	fatal(err)

	// send ᵮ1.00
	resp, err := api.SignPushActions(context.Background(), fio.NewTransferTokensPubKey(account.Actor, to, fio.Tokens(1.0)))
	fatal(err)

	// print the result
	j, err := json.MarshalIndent(resp, "", "  ")
	fatal(err)
	fmt.Println(string(j))
}

```

