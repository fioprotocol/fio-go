package sudo

import eos "github.com/fioprotocol/fio-go/imports/eos-go"

func init() {
	eos.RegisterAction(AN("eosio.wrap"), ActN("exec"), Exec{})
}

var AN = eos.AN
var ActN = eos.ActN
