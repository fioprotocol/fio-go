package system

import (
	"github.com/fioprotocol/fio-go/imports/eos-fio"
)

// NewNonce returns a `nonce` action that lives on the
// `eosio.bios` contract. It should exist only when booting a new
// network, as it is replaced using the `eos-bios` boot process by the
// `eosio.system` contract.
func NewNonce(nonce string) *fos.Action {
	a := &fos.Action{
		Account:       AN("eosio"),
		Name:          ActN("nonce"),
		Authorization: []fos.PermissionLevel{
			//{Actor: AN("eosio"), Permission: PN("active")},
		},
		ActionData: fos.NewActionData(Nonce{
			Value: nonce,
		}),
	}
	return a
}
