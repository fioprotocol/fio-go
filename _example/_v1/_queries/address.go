package main

import (
	"encoding/json"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"github.com/fioprotocol/fio-go/eos"
	"log"
)

func main() {
	const (
		host = "https://testnet.fio.dev"
		address = "cryptolions@fiotestnet"
	)

	e := func(err error) {
		if err != nil {
			log.Fatal(err)
		}
	}

	api, _, err := fio.NewConnection(nil, host)
	e(err)

	// because the fio address is a string, it can only be searched via secondary index. The I128 function implements
	// the hashing algorithm used by the index.
	hashed := fio.I128Hash(address)

	// in this case we'll create our own struct to only extract the account
	type onlyAccount struct {
		OwnerAccount string `json:"owner_account"`
	}

	getRows, err := api.GetTableRows(eos.GetTableRowsRequest{
		Code:       "fio.address",
		Scope:      "fio.address",
		Table:      "fionames",
		LowerBound: hashed,
		UpperBound: hashed,
		KeyType:    "i128",
		Index:      "5",
		JSON:       true,
	})
	e(err)

	// the result is a slice of json.RawMessage, so to deserialize, create a slice of our struct
	rows := make([]onlyAccount, 0)
	err = json.Unmarshal(getRows.Rows, &rows)
	e(err)

	if len(rows) > 0 {
		fmt.Println(address, "is owned by", rows[0].OwnerAccount)
	}
	
}

