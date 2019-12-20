// Copyright 2019 the FIO Foundation.
// Released under the MIT license. See LICENSE for more information.

/*
package fio is for interacting with the FIO Protocol (https://fio.foundation). FIO is the foundation for inter-wallet
operability. An EOS-based blockchain to enhance UX by simplifying naming conventions across multiple chains. This library
heavily relies upon the fantastic eos-go library from EOS Canada. Much gratitude for such a capable underpinning to this
project.


This library primarily extends the eos-go library to provide access to FIO-specific functionality. See the godoc examples
for in-depth coverage.

 $ go get github.com/dapixio/fio-go

Minimal example: equivalent of "cleos get info" (note no error handling for brevity)

 package main

 import (
 	"encoding/json"
 	"fmt"
 	"github.com/dapixio/fio-go"
 )

 func main() {
 	api, _, _ := fio.NewConnection(nil, "https://testnet.fioprotocol.io")
 	info, _ := api.GetInfo()
 	j, _ := json.MarshalIndent(info, "", "  ")
 	fmt.Println(string(j))
 }

*/
package fio
