package fio

import (
	"fmt"
	"log"
)

// this example prints out various chain information
func ExampleAPI_GetCurrentBlock() {

	// Setup a connection, with no credentials associated
	api, _, err := NewConnection(nil, "https://testnet.fioprotocol.io")
	if err != nil {
		log.Fatal("new: " + err.Error())
	}

	// Current top block:
	fmt.Printf("Current block number is: %d\n\n", api.GetCurrentBlock())
}

func ExampleAPI_GetFioProducers() {
	// Setup a connection, with no credentials associated
	api, _, err := NewConnection(nil, "https://testnet.fioprotocol.io")
	if err != nil {
		log.Fatal("new: " + err.Error())
	}

	// Block producer information
	producers, err := api.GetFioProducers()
	if err != nil {
		log.Fatal("producers: " + err.Error())
	}
	if len(producers.Producers) >= 3 {
		fmt.Println("current top 3 block producers by vote:")
		for i := 0; i < 3; i++ {
			fmt.Println("\t", producers.Producers[i].FioAddress)
		}
		fmt.Println("")
	}
}
