package sudo

import "github.com/fioprotocol/fio-go/eos"

func init() {
	eos.RegisterAction(AN("eosio.wrap"), ActN("exec"), Exec{})
}

var AN = eos.AN
var ActN = eos.ActN
