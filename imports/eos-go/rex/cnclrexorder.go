package rex

import (
	eos "github.com/fioprotocol/fio-go/imports/eos-go"
)

func NewCancelREXOrder(owner eos.AccountName) *eos.Action {
	return &eos.Action{
		Account: REXAN,
		Name:    ActN("cnclrexorder"),
		Authorization: []eos.PermissionLevel{
			{Actor: owner, Permission: eos.PermissionName("active")},
		},
		ActionData: eos.NewActionData(CancelREXOrder{
			Owner: owner,
		}),
	}
}

type CancelREXOrder struct {
	Owner eos.AccountName
}
