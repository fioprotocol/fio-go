package main

import (
	"fmt"
	"github.com/fioprotocol/fio-go"
	"log"
	"time"
)

/*
example of getting transactions from get_block. Using this method does not require a history node, but will not
provide full action traces. There are two major downsides to this:

  1. It is not possible to get the fees assessed for a transaction, fees are internal actions and only
     included in an action trace.
  2. Multi-sig transactions are not available, and appear with only the txid where the packed transaction would
     normally exist, this will cause deserialization errors.

This isn't a recommended way of indexing transactions, it will create considerable load on a server, and should only be
done over a local connection. See README.md for more details.

*/

const nodeos = "http://testnet:8888"

func main() {

	// error helper
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	e := func(err error) {
		if err != nil {
			trace := log.Output(2, err.Error())
			log.Fatal(trace)
		}
	}

	// connect
	api, _, err := fio.NewConnection(nil, nodeos)
	e(err)

	// get *all* known ABIs for decoding action data, this is a shortcut providing a map of all current ABIs
	// this is only possible because FIO is not a general-purpose smart-contract platform and only has
	// eight ABIs defined.
	abis, err := api.AllABIs()
	e(err)

	gi, err := api.GetInfo()
	e(err)

	// tracks current block number
	blockNum := gi.LastIrreversibleBlockNum

	// ensure only reversible blocks are printed, track lib in a go routine.
	lib := gi.LastIrreversibleBlockNum
	go func() {
		tick := time.NewTicker(6 * time.Second)
		for {
			select {
			case <-tick.C:
				gi, err := api.GetInfo()
				e(err)
				lib = gi.LastIrreversibleBlockNum
			}
		}
	}()

	// loop over irreversible blocks, print each transaction, and it's actions
	tick := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-tick.C:

			// wait if not a finalized block:
			if blockNum > lib {
				continue
			}

			// fetch the raw block
			block, err := api.GetBlockByNum(blockNum)
			e(err)

			if len(block.Transactions) > 0 {
				log.Println(block.BlockNum, block.ID.String())

				// loop over each transaction
				for _, tx := range block.Transactions {

					// each transaction is packed, unpack it:
					unpacked, err := tx.Transaction.Packed.Unpack()
					e(err)

					// print the transaction
					fmt.Println(unpacked.String())

					// the action data is abi encoded, use the map to decode each action to json in the tx:
					for _, action := range unpacked.Actions {
						act, err := abis[action.Account].DecodeAction(action.HexData, action.Name)
						e(err)
						fmt.Println(string(act))
					}

					fmt.Println("")
				}
			}
			blockNum += 1
		}
	}

}
