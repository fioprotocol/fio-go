package system

import (
	"github.com/fioprotocol/fio-go/imports/eos-fio"
	"github.com/fioprotocol/fio-go/imports/eos-fio/ecc"
)

// NewNewAccount returns a `newaccount` action that lives on the
// `eosio.system` contract.
func NewNewAccount(creator, newAccount fos.AccountName, publicKey fecc.PublicKey) *fos.Action {
	return &fos.Action{
		Account: AN("eosio"),
		Name:    ActN("newaccount"),
		Authorization: []fos.PermissionLevel{
			{Actor: creator, Permission: PN("active")},
		},
		ActionData: fos.NewActionData(NewAccount{
			Creator: creator,
			Name:    newAccount,
			Owner: fos.Authority{
				Threshold: 1,
				Keys: []fos.KeyWeight{
					{
						PublicKey: publicKey,
						Weight:    1,
					},
				},
				Accounts: []fos.PermissionLevelWeight{},
			},
			Active: fos.Authority{
				Threshold: 1,
				Keys: []fos.KeyWeight{
					{
						PublicKey: publicKey,
						Weight:    1,
					},
				},
				Accounts: []fos.PermissionLevelWeight{},
			},
		}),
	}
}

// NewDelegatedNewAccount returns a `newaccount` action that lives on the
// `eosio.system` contract. It is filled with an authority structure that
// delegates full control of the new account to an already existing account.
func NewDelegatedNewAccount(creator, newAccount fos.AccountName, delegatedTo fos.AccountName) *fos.Action {
	return &fos.Action{
		Account: AN("eosio"),
		Name:    ActN("newaccount"),
		Authorization: []fos.PermissionLevel{
			{Actor: creator, Permission: PN("active")},
		},
		ActionData: fos.NewActionData(NewAccount{
			Creator: creator,
			Name:    newAccount,
			Owner: fos.Authority{
				Threshold: 1,
				Keys:      []fos.KeyWeight{},
				Accounts: []fos.PermissionLevelWeight{
					fos.PermissionLevelWeight{
						Permission: fos.PermissionLevel{
							Actor:      delegatedTo,
							Permission: PN("active"),
						},
						Weight: 1,
					},
				},
			},
			Active: fos.Authority{
				Threshold: 1,
				Keys:      []fos.KeyWeight{},
				Accounts: []fos.PermissionLevelWeight{
					fos.PermissionLevelWeight{
						Permission: fos.PermissionLevel{
							Actor:      delegatedTo,
							Permission: PN("active"),
						},
						Weight: 1,
					},
				},
			},
		}),
	}
}

// NewCustomNewAccount returns a `newaccount` action that lives on the
// `eosio.system` contract. You can specify your own `owner` and
// `active` permissions.
func NewCustomNewAccount(creator, newAccount fos.AccountName, owner, active fos.Authority) *fos.Action {
	return &fos.Action{
		Account: AN("eosio"),
		Name:    ActN("newaccount"),
		Authorization: []fos.PermissionLevel{
			{Actor: creator, Permission: PN("active")},
		},
		ActionData: fos.NewActionData(NewAccount{
			Creator: creator,
			Name:    newAccount,
			Owner:   owner,
			Active:  active,
		}),
	}
}

// NewAccount represents a `newaccount` action on the `eosio.system`
// contract. It is one of the rare ones to be hard-coded into the
// blockchain.
type NewAccount struct {
	Creator fos.AccountName `json:"creator"`
	Name    fos.AccountName `json:"name"`
	Owner   fos.Authority   `json:"owner"`
	Active  fos.Authority   `json:"active"`
}
