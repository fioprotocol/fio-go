package token

import "github.com/fioprotocol/fio-go/imports/eos-fio"

func init() {
	feos.RegisterAction(AN("eosio.token"), ActN("transfer"), Transfer{})
	feos.RegisterAction(AN("eosio.token"), ActN("issue"), Issue{})
	feos.RegisterAction(AN("eosio.token"), ActN("create"), Create{})
}
