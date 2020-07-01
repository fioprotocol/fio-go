package system

import (
	"github.com/fioprotocol/fio-go/imports/eos-fio"
)

// NewLinkAuth creates an action from the `eosio.system` contract
// called `linkauth`.
//
// `linkauth` allows you to attach certain permission to the given
// `code::actionName`. With this set on-chain, you can use the
// `requiredPermission` to sign transactions for `code::actionName`
// and not rely on your `active` (which might be more sensitive as it
// can sign anything) for the given operation.
func NewLinkAuth(account, code fos.AccountName, actionName fos.ActionName, requiredPermission fos.PermissionName) *fos.Action {
	a := &fos.Action{
		Account: AN("eosio"),
		Name:    ActN("linkauth"),
		Authorization: []fos.PermissionLevel{
			{
				Actor:      account,
				Permission: fos.PermissionName("active"),
			},
		},
		ActionData: fos.NewActionData(LinkAuth{
			Account:     account,
			Code:        code,
			Type:        actionName,
			Requirement: requiredPermission,
		}),
	}

	return a
}

// LinkAuth represents the native `linkauth` action, through the
// system contract.
type LinkAuth struct {
	Account     fos.AccountName    `json:"account"`
	Code        fos.AccountName    `json:"code"`
	Type        fos.ActionName     `json:"type"`
	Requirement fos.PermissionName `json:"requirement"`
}
