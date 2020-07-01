package token

import "github.com/fioprotocol/fio-go/imports/eos-fio"

func init() {
	fos.RegisterAction(AN("eosio.token"), ActN("transfer"), Transfer{})
	fos.RegisterAction(AN("eosio.token"), ActN("issue"), Issue{})
	fos.RegisterAction(AN("eosio.token"), ActN("create"), Create{})
}
