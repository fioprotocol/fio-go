package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fioprotocol/fio-go/v2"
	"log"
	"time"
)

// simple example to connect and print chain information

func main() {
	const (
		url = "https://testnet.fioprotocol.io"
	)

	// error helper
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	fatal := func(err error) {
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

	// connection, not associated with a private key
	api, _, err := fio.NewConnection(cx(), nil, url)
	fatal(err)

	// print out chain information
	info, err := api.GetInfo(cx())
	fatal(err)

	j, err := json.MarshalIndent(info, "", "  ")
	fatal(err)
	fmt.Println(string(j))
}
