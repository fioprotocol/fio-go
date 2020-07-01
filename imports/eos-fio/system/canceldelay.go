package system

import (
	"github.com/fioprotocol/fio-go/imports/eos-fio"
)

// NewCancelDelay creates an action from the `eosio.system` contract
// called `canceldelay`.
//
// `canceldelay` allows you to cancel a deferred transaction,
// previously sent to the chain with a `delay_sec` larger than 0.  You
// need to sign with cancelingAuth, to cancel a transaction signed
// with that same authority.
func NewCancelDelay(cancelingAuth fos.PermissionLevel, transactionID fos.Checksum256) *fos.Action {
	a := &fos.Action{
		Account: AN("eosio"),
		Name:    ActN("canceldelay"),
		Authorization: []fos.PermissionLevel{
			cancelingAuth,
		},
		ActionData: fos.NewActionData(CancelDelay{
			CancelingAuth: cancelingAuth,
			TransactionID: transactionID,
		}),
	}

	return a
}

// CancelDelay represents the native `canceldelay` action, through the
// system contract.
type CancelDelay struct {
	CancelingAuth fos.PermissionLevel `json:"canceling_auth"`
	TransactionID fos.Checksum256     `json:"trx_id"`
}
