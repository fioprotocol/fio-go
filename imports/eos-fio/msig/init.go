package msig

import (
	"github.com/fioprotocol/fio-go/imports/eos-fio"
)

func init() {
	fos.RegisterAction(AN("eosio.msig"), ActN("propose"), &Propose{})
	fos.RegisterAction(AN("eosio.msig"), ActN("approve"), &Approve{})
	fos.RegisterAction(AN("eosio.msig"), ActN("unapprove"), &Unapprove{})
	fos.RegisterAction(AN("eosio.msig"), ActN("cancel"), &Cancel{})
	fos.RegisterAction(AN("eosio.msig"), ActN("exec"), &Exec{})
}

var AN = fos.AN
var PN = fos.PN
var ActN = fos.ActN
