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

/*
TODO: use reflection to allow setting the Tpid in an Action if the field exists:
// Action is a clone of eos.Action so it can have custom member functions
type Action eos.Action

func (a *Action) SetTpid(tpid string) error {
	actionType := reflect.TypeOf(a.ActionData.Data)
	value, ok := actionType.FieldByName(`Tpid`)
	if !ok {
		return errors.New("transaction does not contain a tpid field")
	}
	reflect.ValueOf(value).Set("tpid")
}

 */