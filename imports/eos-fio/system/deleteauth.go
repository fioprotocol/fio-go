package system

import (
	fos "github.com/fioprotocol/fio-go/imports/eos-fio"
)

// NewDeleteAuth creates an action from the `eosio.system` contract
// called `deleteauth`.
//
// You cannot delete the `owner` or `active` permissions.  Also, if a
// permission is still linked through a previous `updatelink` action,
// you will need to `unlinkauth` first.
func NewDeleteAuth(account fos.AccountName, permission fos.PermissionName) *fos.Action {
	a := &fos.Action{
		Account: AN("eosio"),
		Name:    ActN("deleteauth"),
		Authorization: []fos.PermissionLevel{
			{Actor: account, Permission: fos.PermissionName("active")},
		},
		ActionData: fos.NewActionData(DeleteAuth{
			Account:    account,
			Permission: permission,
		}),
	}

	return a
}

// DeleteAuth represents the native `deleteauth` action, reachable
// through the `eosio.system` contract.
type DeleteAuth struct {
	Account    fos.AccountName    `json:"account"`
	Permission fos.PermissionName `json:"permission"`
}
