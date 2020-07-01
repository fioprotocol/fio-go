package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"

	"github.com/fioprotocol/fio-go/imports/eos-fio/fecc"
)

func main() {
	flag.Parse()

	if len(flag.Args()) != 1 {
		log.Fatalln("Please specify a public key")
	}

	textkey := flag.Args()[0]
	pubkey, err := fecc.NewPublicKey(textkey)
	if err != nil {
		log.Fatalln("invalid public key:", err)
	}

	fmt.Printf("public key in hex form: %q\n", hex.EncodeToString(pubkey.Content))
}
