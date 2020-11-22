package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/blockpane/eos-go"
	"github.com/fioprotocol/fio-go/v2"
	"log"
	"time"
)

func main() {
	const (
		host   = "https://testnet.fio.dev"
		domain = "fiotestnet"
	)

	// error helper
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	e := func(err error) {
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

	api, _, err := fio.NewConnection(cx(), nil, host)
	e(err)

	// because the fio domain is a string, it can only be searched via secondary index. The I128 function implements
	// the hashing algorithm used by the index.
	hashed := fio.I128Hash(domain)

	// in this case we'll create our own struct to only extract the account
	type onlyAccount struct {
		Account string `json:"account"`
	}

	getRows, err := api.GetTableRows(cx(), eos.GetTableRowsRequest{
		Code:       "fio.address",
		Scope:      "fio.address",
		Table:      "domains",
		LowerBound: hashed,
		UpperBound: hashed,
		KeyType:    "i128",
		Index:      "4",
		JSON:       true,
	})
	e(err)

	// the result is a slice of json.RawMessage, so to deserialize, create a slice of our struct
	result := make([]onlyAccount, 0)
	err = json.Unmarshal(getRows.Rows, &result)
	e(err)

	if len(result) > 0 {
		fmt.Println(domain, "is owned by", result[0].Account)
	}

}

