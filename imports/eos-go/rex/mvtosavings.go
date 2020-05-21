package rex

import (
	eos "github.com/fioprotocol/fio-go/imports/eos-go"
)

func NewMoveToSavings(owner eos.AccountName, rex eos.Asset) *eos.Action {
	return &eos.Action{
		Account: REXAN,
		Name:    ActN("mvtosavings"),
		Authorization: []eos.PermissionLevel{
			{Actor: owner, Permission: eos.PermissionName("active")},
		},
		ActionData: eos.NewActionData(MoveToSavings{
			Owner: owner,
			REX:   rex,
		}),
	}
}

type MoveToSavings struct {
	Owner eos.AccountName
	REX   eos.Asset
}
