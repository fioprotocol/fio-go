package system

import (
	fos "github.com/fioprotocol/fio-go/imports/eos-fio"
)

// NewUnlinkAuth creates an action from the `eosio.system` contract
// called `unlinkauth`.
//
// `unlinkauth` detaches a previously set permission from a
// `code::actionName`. See `linkauth`.
func NewUnlinkAuth(account, code fos.AccountName, actionName fos.ActionName) *fos.Action {
	a := &fos.Action{
		Account: AN("eosio"),
		Name:    ActN("unlinkauth"),
		Authorization: []fos.PermissionLevel{
			{
				Actor:      account,
				Permission: fos.PermissionName("active"),
			},
		},
		ActionData: fos.NewActionData(UnlinkAuth{
			Account: account,
			Code:    code,
			Type:    actionName,
		}),
	}

	return a
}

// UnlinkAuth represents the native `unlinkauth` action, through the
// system contract.
type UnlinkAuth struct {
	Account fos.AccountName `json:"account"`
	Code    fos.AccountName `json:"code"`
	Type    fos.ActionName  `json:"type"`
}
