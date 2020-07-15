package msig

import (
	"github.com/fioprotocol/fio-go/imports/eos-fio"
)

func init() {
	feos.RegisterAction(AN("eosio.msig"), ActN("propose"), &Propose{})
	feos.RegisterAction(AN("eosio.msig"), ActN("approve"), &Approve{})
	feos.RegisterAction(AN("eosio.msig"), ActN("unapprove"), &Unapprove{})
	feos.RegisterAction(AN("eosio.msig"), ActN("cancel"), &Cancel{})
	feos.RegisterAction(AN("eosio.msig"), ActN("exec"), &Exec{})
}

var AN = feos.AN
var PN = feos.PN
var ActN = feos.ActN
