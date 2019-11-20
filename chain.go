package fio

import (
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
)

// NewConnection sets up the eos.API interface for interacting with the FIO API
func NewConnection(keyBag *eos.KeyBag, url string) (*eos.API, *eos.TxOptions, error) {
	api := eos.New(url)
	api.SetSigner(keyBag)
	api.SetCustomGetRequiredKeys(
		func(tx *eos.Transaction) (keys []ecc.PublicKey, e error) {
			return keyBag.AvailableKeys()
		},
	)
	txOpts := &eos.TxOptions{}
	if err := txOpts.FillFromChain(api); err != nil {
		return nil, nil, err
	}
	return api, txOpts, nil
}

// newAction creates an eos.Action for FIO contract calls
func newAction(contract eos.AccountName, name eos.ActionName, actor eos.AccountName, actionData interface{}) *eos.Action {
	return &eos.Action{
		Account:       contract,
		Name:          name,
		Authorization: []eos.PermissionLevel{
			{
				Actor:      actor,
				Permission: "active",
			},
		},
		ActionData:    eos.NewActionData(actionData),
	}
}
