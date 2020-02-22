package fio

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eoscanada/eos-go"
	"math"
	"sort"
	"strconv"
	"time"
)

// PermissionLevel wraps eos.PermissionLevel to add a convenience function
type PermissionLevel eos.PermissionLevel

func NewPermissionLevel(account eos.AccountName) *PermissionLevel {
	return &PermissionLevel{
		Actor:      account,
		Permission: "active", // Permission is always active on FIO chain
	}
}

func NewPermissionLevelSlice(accounts []string) []*PermissionLevel {
	l := make([]*PermissionLevel, 0)
	sort.Strings(accounts)
	for _, a := range accounts {
		l = append(l, NewPermissionLevel(eos.AccountName(a)))
	}
	return l
}

func (pl PermissionLevel) ToEos() *eos.PermissionLevel {
	return &eos.PermissionLevel{
		Actor:      pl.Actor,
		Permission: pl.Permission,
	}
}

type MsigAction struct {
	Account       eos.Name        `json:"account"`
	Name          eos.Name        `json:"name"`
	Authorization PermissionLevel `json:"authorization"`
	Data          []byte          `json:"data"`
}

type MsigApproval struct {
	Level PermissionLevel `json:"level"`
	Time  eos.JSONTime    `json:"time"`
}

type MsigApprovalsInfo struct {
	Version            uint8          `json:"version"`
	ProposalName       eos.Name       `json:"proposal_name"`
	RequestedApprovals []MsigApproval `json:"requested_approvals"`
	ProvidedApprovals  []MsigApproval `json:"provided_approvals"`
}

func (api *API) GetApprovals(scope Name) (more bool, info []*MsigApprovalsInfo, err error) {
	name, err := eos.StringToName(string(scope))
	if err != nil {
		return false, nil, err
	}
	res, err := api.GetTableRows(eos.GetTableRowsRequest{
		JSON:       true,
		Scope:      fmt.Sprintf("%d", name),
		Code:       "eosio.msig",
		Table:      "approvals2",
		Limit:      math.MaxUint32,
	})
	if err != nil {
		return false, nil, err
	}
	more = res.More
	info = make([]*MsigApprovalsInfo, 0)
	err = json.Unmarshal(res.Rows, &info)
	return
}

func (info MsigApprovalsInfo) HasRequested(actor eos.AccountName) bool {
	for _, r := range info.RequestedApprovals {
		if r.Level.Actor == actor {
			return true
		}
	}
	for _, p := range info.ProvidedApprovals {
		if p.Level.Actor == actor {
			return true
		}
	}
	return false
}

/*
// TODO: This looks suspiciously incorrect, we can probably replace PackedTransaction with an eos.Transaction and
// bypass a step when unmarshalling
type MsigProposal struct {
	ProposalName     eos.Name `json:"proposal_name"`
	PackedTransaction []byte   `json:"packed_trasaction"`
}
*/

// TODO: This smells like there is another intermediate type involved, or an enum that needs to be included here
//TODO: read up on extensions and ensure this is idiomatic
type MsigExtension struct {
	Type uint16 `json:"type"`
	Data []byte `json:"data"`
}

type MsigInvalidation struct {
	Account              eos.Name      `json:"account"`
	LastInvalidationTime eos.TimePoint `json:"last_invalidation_time"`
}

type MsigOldApprovalsInfo struct {
	ProposalName       eos.Name          `json:"proposal_name"`
	RequestedApprovals []PermissionLevel `json:"requested_approvals"`
	ProvidedApprovals  []PermissionLevel `json:"provided_approvals"`
}

// this also looks potentially incorrect:
type MsigTransaction struct {
	ContextFreeActions    []*Action   `json:"context_free_actions"`
	Actions               []*Action   `json:"actions"`
	TransactionExtensions []*MsigExec `json:"transaction_extensions"`
}

// MsigTransactionHeader is an alias for consistent naming
type MsigTransactionHeader eos.TransactionHeader

/*

Actions

*/

type MsigApprove struct {
	Proposer     eos.AccountName `json:"proposer"`
	ProposalName eos.Name        `json:"proposal_name"`
	Level        PermissionLevel `json:"level"`
	MaxFee       uint64          `json:"max_fee"`
	ProposalHash eos.Checksum256 `json:"proposal_hash"`
}

type MsigCancel struct {
	Proposer     eos.AccountName `json:"proposer"`
	ProposalName eos.Name        `json:"proposal_name"`
	Canceler     eos.AccountName `json:"canceler"`
	MaxFee       uint64          `json:"max_fee"`
}

type MsigExec struct {
	Proposer     eos.AccountName `json:"proposer"`
	ProposalName eos.Name        `json:"proposal_name"`
	MaxFee       uint64          `json:"max_fee"`
	Executer     eos.Name        `json:"executer"`
}

type MsigInvalidate struct {
	Name   eos.Name `json:"name"`
	MaxFee uint64   `json:"max_fee"`
}

type MsigPropose struct {
	Proposer     eos.AccountName        `json:"proposer"`
	ProposalName eos.Name               `json:"proposal_name"`
	Requested    []*PermissionLevel     `json:"requested"`
	MaxFee       uint64                 `json:"max_fee"`
	Trx          *eos.SignedTransaction `json:"trx"`
}

// NewMsigPropose is provided for consistency, but it will make more sense to use NewSignedMsigPropose to build multisig proposals since it
// abstracts several steps. Note that the []PermissionLever.
func NewMsigPropose(proposer eos.AccountName, proposal eos.Name, signers []*PermissionLevel, signedTx *eos.SignedTransaction) *Action {
	var feeBytes uint64
	packedTx, err := signedTx.Pack(CompressionNone)
	if err != nil {
		feeBytes = 1
	} else {
		feeBytes = uint64((len(packedTx.PackedTransaction) / 1000) + 1)
	}

	return NewAction("eosio.msig", "propose", proposer, MsigPropose{
		Proposer:     proposer,
		ProposalName: proposal,
		Requested:    signers,
		MaxFee:       Tokens(GetMaxFee(FeeMsigPropose)) * feeBytes,
		Trx:          signedTx,
	})
}

// NewSignedMsigPropose simplifies the process of building an MsigPropose by packing and signing the slice of Actions provided into a TX
// and then wrapping that into a signed transaction ready to be submitted.
func (api *API) NewSignedMsigPropose(proposalName Name, approvers []string, actions []*Action, expires time.Duration, signer *Account, txOpt *TxOptions) (*eos.PackedTransaction, error) {
	if len(actions) == 0 {
		return nil, errors.New("no actions provided")
	}
	if signer == nil || signer.KeyBag == nil || len(signer.KeyBag.Keys) == 0 {
		return nil, errors.New("invalid signer, no private key provided")
	}
	for _, apvr := range approvers {
		if len(apvr) > 12 {
			return nil, errors.New("invalid approver in list, account name should be < 12 chars")
		}
	}
	propTx := NewTransaction(actions, txOpt)
	propTx.Expiration = eos.JSONTime{Time: time.Now().UTC().Add(expires)}
	propTxSigned, propTxPacked, err := api.SignTransaction(propTx, txOpt.ChainID, CompressionNone)
	if err != nil {
		return nil, err
	}
	feeBytes := uint64((len(propTxPacked.PackedTransaction) / 1000) + 1)

	newTx := NewTransaction([]*Action{NewAction(
		"eosio.msig", "propose", signer.Actor, MsigPropose{
			Proposer:     signer.Actor,
			ProposalName: proposalName.ToEos(),
			Requested:    NewPermissionLevelSlice(approvers),
			MaxFee:       Tokens(GetMaxFee(FeeMsigPropose)) * feeBytes,
			Trx:          propTxSigned,
		},
	)}, txOpt)
	newTx.Expiration = eos.JSONTime{Time: time.Now().UTC().Add(expires)}
	_, packedTx, err := api.SignTransaction(newTx, txOpt.ChainID, CompressionZlib)
	if err != nil {
		return nil, err
	}

	return packedTx, nil
}

type MsigUnapprove struct {
	Proposer     eos.AccountName `json:"proposer"`
	ProposalName eos.Name        `json:"proposal_name"`
	Level        PermissionLevel `json:"level"`
	MaxFee       uint64          `json:"max_fee"`
}

type Authority eos.Authority

type UpdateAuth struct {
	Account    eos.AccountName `json:"account"`
	Permission eos.Name        `json:"permission"`
	Parent     eos.Name        `json:"parent"`
	Auth       Authority       `json:"auth"`
	MaxFee     uint64          `json:"max_fee"`
}

// NewUpdateAuthSimple just takes a list of accounts and a threshold. Nothing fancy, most basic EOS msig account.
func NewUpdateAuthSimple(account eos.AccountName, actors []string, threshold uint32) *Action {
	acts := make([]eos.PermissionLevelWeight, 0)
	sort.Strings(actors) // actors must be sorted in ascending alphabetic order, or will get an invalid {$auth} error.
	for _, a := range actors {
		acts = append(acts, eos.PermissionLevelWeight{
			Weight:     1,
			Permission: eos.PermissionLevel{Actor: eos.AccountName(a), Permission: "active"}})
	}
	return NewAction("eosio", "updateauth", eos.AccountName(account), UpdateAuth{
		Account:    account,
		Permission: "active",
		Parent:     "owner",
		Auth: Authority{
			Threshold: threshold,
			Accounts:  acts,
		},
		MaxFee: Tokens(GetMaxFee(FeeAuthUpdate)),
	})
}

type scopeResp struct {
	Scope string `json:"scope"`
	Count int    `json:"count"`
}

// GetProposals fetches the proposal list from eosio.msig returning a map of scopes, with a count for each
func (api *API) GetProposals(offset int, limit int) (more bool, scopes map[string]int, err error) {
	res, err := api.GetTableByScopeMore(eos.GetTableByScopeRequest{
		Code:       "eosio.msig",
		Table:      "proposal",
		LowerBound: strconv.Itoa(offset),
		UpperBound: "",
		Limit:      uint32(limit),
	})
	if err != nil {
		return false, nil, err
	}
	more = res.More
	resScopes := make([]scopeResp, 0)
	err = json.Unmarshal(res.Rows, &resScopes)
	if err != nil {
		return false, nil, err
	}
	scopes = make(map[string]int)
	for _, s := range resScopes {
		scopes[s.Scope] = s.Count
	}
	return
}

