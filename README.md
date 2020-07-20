# FIO-GO

Library for interacting with the FIO network using the go language.

## Breaking Changes

In 1.0.0 and later eos-go has been imported, this is to facilitate ECC changes needed for FIO and to ensure API stability.
Updating existing code using eos-go dependencies should only require:

```
import (
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
)
```

becomes:

```
import (
	"github.com/fioprotocol/fio-go/eos"
	"github.com/fioprotocol/fio-go/eos/ecc"
)
```

## Example

This demonstrates using the library to send FIO tokens from one account to another:

```go
package main

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
		to = `FIO6G9pXXM92Gy5eMwNquGULoCj3ZStwPLPdEb9mVXyEHqWN7HSuA`
	)

	fatal := func(e error) {
    	if e != nil {
    		log.Fatal(e)
    	}
    }

	// connect to the network, using credentials
	account, api, _, err := fio.NewWifConnect(wif, url)
	fatal(err)

	// send ᵮ1.00
	resp, err := api.SignPushActions(fio.NewTransferTokensPubKey(account.Actor, to, fio.Tokens(1.0)))
	fatal(err)

	// print the result
	j, err := json.MarshalIndent(resp, "", "  ")
	fatal(err)
	fmt.Println(string(j))
}

```

