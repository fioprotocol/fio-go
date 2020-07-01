package system

import (
	fos "github.com/fioprotocol/fio-go/imports/eos-fio"
)

// NewNonce returns a `nonce` action that lives on the
// `eosio.bios` contract. It should exist only when booting a new
// network, as it is replaced using the `eos-bios` boot process by the
// `eosio.system` contract.
func NewVoteProducer(voter fos.AccountName, proxy fos.AccountName, producers ...fos.AccountName) *fos.Action {
	a := &fos.Action{
		Account: AN("eosio"),
		Name:    ActN("voteproducer"),
		Authorization: []fos.PermissionLevel{
			{Actor: voter, Permission: PN("active")},
		},
		ActionData: fos.NewActionData(
			VoteProducer{
				Voter:     voter,
				Proxy:     proxy,
				Producers: producers,
			},
		),
	}
	return a
}

// VoteProducer represents the `eosio.system::voteproducer` action
type VoteProducer struct {
	Voter     fos.AccountName   `json:"voter"`
	Proxy     fos.AccountName   `json:"proxy"`
	Producers []fos.AccountName `json:"producers"`
}
