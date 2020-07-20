package main

import (
	"encoding/hex"
	"fmt"
	"github.com/fioprotocol/fio-go/eos"
	"log"
	"os"

	"github.com/davecgh/go-spew/spew"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("missing transaction hex as argument")
	}
	fmt.Printf("Enter the raw transaction as hex: ")

	fmt.Println("STRING", os.Args[1])

	b, err := hex.DecodeString(os.Args[1])
	if err != nil {
		log.Fatalln("error decoding hex:", err)
	}

	var tx *eos.Transaction
	err = eos.UnmarshalBinary(b, &tx)
	if err != nil {
		log.Fatalln("error decoding:", err)
	}

	spew.Dump(tx)
}
