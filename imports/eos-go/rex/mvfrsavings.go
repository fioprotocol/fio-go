package rex

import (
	eos "github.com/fioprotocol/fio-go/imports/eos-go"
)

func NewMoveFromSavings(owner eos.AccountName, rex eos.Asset) *eos.Action {
	return &eos.Action{
		Account: REXAN,
		Name:    ActN("mvfrsavings"),
		Authorization: []eos.PermissionLevel{
			{Actor: owner, Permission: eos.PermissionName("active")},
		},
		ActionData: eos.NewActionData(MoveFromSavings{
			Owner: owner,
			REX:   rex,
		}),
	}
}

type MoveFromSavings struct {
	Owner eos.AccountName
	REX   eos.Asset
}
