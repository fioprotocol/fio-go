package main

import (
	"encoding/json"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"log"
)

// simple example to connect and print chain information

func main() {
	const (
		url = "https://testnet.fioprotocol.io"
	)

	fatal := func(e error) {
		if e != nil {
			log.Fatal(e)
		}
	}

	// connection, not associated with a private key
	api, _, err := fio.NewConnection(nil, url)
	fatal(err)

	// print out chain information
	info, err := api.GetInfo()
	fatal(err)

	j, err := json.MarshalIndent(info, "", "  ")
	fatal(err)
	fmt.Println(string(j))
}
