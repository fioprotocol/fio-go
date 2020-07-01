package system

import (
	fos "github.com/fioprotocol/fio-go/imports/eos-fio"
)

// NewUpdateAuth creates an action from the `eosio.system` contract
// called `updateauth`.
//
// usingPermission needs to be `owner` if you want to modify the
// `owner` authorization, otherwise `active` will do for the rest.
func NewUpdateAuth(account fos.AccountName, permission, parent fos.PermissionName, authority fos.Authority, usingPermission fos.PermissionName) *fos.Action {
	a := &fos.Action{
		Account: AN("eosio"),
		Name:    ActN("updateauth"),
		Authorization: []fos.PermissionLevel{
			{
				Actor:      account,
				Permission: usingPermission,
			},
		},
		ActionData: fos.NewActionData(UpdateAuth{
			Account:    account,
			Permission: permission,
			Parent:     parent,
			Auth:       authority,
		}),
	}

	return a
}

// UpdateAuth represents the hard-coded `updateauth` action.
//
// If you change the `active` permission, `owner` is the required parent.
//
// If you change the `owner` permission, there should be no parent.
type UpdateAuth struct {
	Account    fos.AccountName    `json:"account"`
	Permission fos.PermissionName `json:"permission"`
	Parent     fos.PermissionName `json:"parent"`
	Auth       fos.Authority      `json:"auth"`
}
