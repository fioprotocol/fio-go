package main

import (
	"encoding/json"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"github.com/fioprotocol/fio-go/eos"
	"log"
)

// example of voting for fees and setting a multiplier

func main() {

	const (
		url = `http://dev:8888`
		wif = `5JP1fUXwPxuKuNryh5BEsFhZqnh59yVtpHqHxMMTmtjcni48bqC`
	)

	// error helper
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	fatal := func(err error) {
		if err != nil {
			trace := log.Output(2, err.Error())
			log.Fatal(trace)
		}
	}

	account, api, opts, err := fio.NewWifConnect(wif, url)
	fatal(err)

	action := fio.NewSetFeeVote(defaultRatios(), account.Actor).ToEos() // note casting to *eos.Action

	// this is a large tx, without compression it might fail
	opts.Compress = fio.CompressionZlib
	// overriding the default compression requires a using different function
	resp, err := api.SignPushActionsWithOpts([]*eos.Action{action}, &opts.TxOptions)
	fatal(err)

	// print result
	j, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(j))

	// Now set the fee multiplier
	var (
		tokenPriceUsd      float64 = 0.08                       // for the example assume 1 FIO is worth 8 cents
		targetUsd          float64 = 2.00                       // and the goal is for regaddress to cost $2.00
		regaddressFeeValue float64 = 2000000000 / 1_000_000_000 // and the current fee value is set to 2 FIO (in SUF)
	)

	// 12.5
	multiplier := targetUsd / (regaddressFeeValue * tokenPriceUsd)

	// submit and print the result
	resp, err = api.SignPushActions(fio.NewSetFeeMult(multiplier, account.Actor))
	fatal(err)
	j, _ = json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(j))

	// it's also important that computefees is called frequently, the on-chain fees don't change automatically without it
	// this call won't always have any work to do, so it's safe to ignore errors.
	resp, err = api.SignPushActions(fio.NewComputeFees(account.Actor))
	if err != nil {
		log.Println(err)
	}
	j, _ = json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(j))

}

// defaultRatios should be values originally set for each action.
func defaultRatios() []*fio.FeeValue {
	return []*fio.FeeValue{
		{
			EndPoint: "register_fio_domain",
			Value:    40000000000,
		},
		{
			EndPoint: "register_fio_address",
			Value:    2000000000,
		},
		{
			EndPoint: "renew_fio_domain",
			Value:    40000000000,
		},
		{
			EndPoint: "renew_fio_address",
			Value:    2000000000,
		},
		{
			EndPoint: "add_pub_address",
			Value:    30000000,
		},
		{
			EndPoint: "transfer_tokens_pub_key",
			Value:    100000000,
		},
		{
			EndPoint: "new_funds_request",
			Value:    60000000,
		},
		{
			EndPoint: "reject_funds_request",
			Value:    30000000,
		},
		{
			EndPoint: "record_obt_data",
			Value:    60000000,
		},
		{
			EndPoint: "set_fio_domain_public",
			Value:    30000000,
		},
		{
			EndPoint: "register_producer",
			Value:    10000000000,
		},
		{
			EndPoint: "register_proxy",
			Value:    1000000000,
		},
		{
			EndPoint: "unregister_proxy",
			Value:    20000000,
		},
		{
			EndPoint: "unregister_producer",
			Value:    20000000,
		},
		{
			EndPoint: "proxy_vote",
			Value:    30000000,
		},
		{
			EndPoint: "vote_producer",
			Value:    30000000,
		},
		{
			EndPoint: "add_to_whitelist",
			Value:    30000000,
		},
		{
			EndPoint: "remove_from_whitelist",
			Value:    30000000,
		},
		{
			EndPoint: "submit_bundled_transaction",
			Value:    30000000,
		},
		{
			EndPoint: "auth_delete",
			Value:    20000000,
		},
		{
			EndPoint: "auth_link",
			Value:    20000000,
		},
		{
			EndPoint: "auth_update",
			Value:    50000000,
		},
		{
			EndPoint: "msig_propose",
			Value:    50000000,
		},
		{
			EndPoint: "msig_approve",
			Value:    20000000,
		},
		{
			EndPoint: "msig_unapprove",
			Value:    20000000,
		},
		{
			EndPoint: "msig_cancel",
			Value:    20000000,
		},
		{
			EndPoint: "msig_exec",
			Value:    20000000,
		},
		{
			EndPoint: "msig_invalidate",
			Value:    20000000,
		},
		{
			EndPoint: "cancel_funds_request",
			Value:    60000000,
		},
		{
			EndPoint: "remove_pub_address",
			Value:    60000000,
		},
		{
			EndPoint: "remove_all_pub_addresses",
			Value:    60000000,
		},
		{
			EndPoint: "transfer_fio_domain",
			Value:    100000000,
		},
		{
			EndPoint: "transfer_fio_address",
			Value:    60000000,
		},
		{
			EndPoint: "submit_fee_multiplier",
			Value:    60000000,
		},
		{
			EndPoint: "submit_fee_ratios",
			Value:    20000000,
		},
		{
			EndPoint: "burn_fio_address",
			Value:    60000000,
		},
	}
}
