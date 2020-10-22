package main

import (
	"encoding/json"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"log"
	"time"
)

/*
example of getting transactions from a v1 (light) history node. This is easier than using get_block, but can also
result in a significant amount of traffic and is only recommended for a local node.

** This requires the history plugin to be enabled in the config.ini file**

    plugin = eosio::history_plugin
    plugin = eosio::history_api_plugin
    filter-on = *
    filter-out = eosio:onblock:
    history-per-account = 100000

Note: Action traces (ie trace.Traces[0].Action.ActionData.Data) are returned as a map[string]interface{}

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

	gi, err := api.GetInfo()
	e(err)

	// tracks current block number
	blockNum := gi.LastIrreversibleBlockNum

	// loop over irreversible blocks, get list of transaction IDs, then fetch and print each transaction
	tick := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-tick.C:
			txids, err := api.HistGetBlockTxids(blockNum)
			e(err)

			// don't process reversible blocks:
			if blockNum > txids.LastIrreversibleBlock {
				continue
			}

			// loop over each transaction and print it including full action trace data
			if len(txids.Ids) > 0 {
				log.Println("block: ", blockNum)
				for _, id := range txids.Ids {
					trace, err := api.GetTransaction(id)
					e(err)

					j, err := json.MarshalIndent(trace, "", "  ")
					e(err)
					fmt.Println(string(j))
				}
			}
		}
		blockNum += 1
	}
}
