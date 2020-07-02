package fio

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	feos "github.com/fioprotocol/fio-go/imports/eos-fio"
	"sort"
	"strconv"
	"time"
)

// PermissionLevel wraps eos-go's type to add member functions
type PermissionLevel feos.PermissionLevel

func NewPermissionLevel(account feos.AccountName) *PermissionLevel {
	return &PermissionLevel{
		Actor:      account,
		Permission: "active",
	}
}

// NewPermissionLevelSlice is a convenience function for quickly building a slice of active permissions
func NewPermissionLevelSlice(accounts []string) []*PermissionLevel {
	l := make([]*PermissionLevel, 0)
	sort.Strings(accounts)
	for _, a := range accounts {
		l = append(l, NewPermissionLevel(feos.AccountName(a)))
	}
	return l
}

// ToEos converts from fio.PermissionLevel to eos.PermissionLevel
func (pl PermissionLevel) ToEos() *feos.PermissionLevel {
	return &feos.PermissionLevel{
		Actor:      pl.Actor,
		Permission: pl.Permission,
	}
}

type MsigAction struct {
	Account       feos.Name       `json:"account"`
	Name          feos.Name       `json:"name"`
	Authorization PermissionLevel `json:"authorization"`
	Data          []byte          `json:"data"`
}

type MsigApproval struct {
	Level PermissionLevel `json:"level"`
	Time  feos.JSONTime   `json:"time"`
}

type MsigApprovalsInfo struct {
	Version            uint8          `json:"version"`
	ProposalName       feos.Name      `json:"proposal_name"`
	RequestedApprovals []MsigApproval `json:"requested_approvals"`
	ProvidedApprovals  []MsigApproval `json:"provided_approvals"`
}

// GetApprovals returns a list of approvals for an account
func (api *API) GetApprovals(scope Name, limit int) (more bool, info []*MsigApprovalsInfo, err error) {
	name, err := feos.StringToName(string(scope))
	if err != nil {
		return false, nil, err
	}
	res, err := api.GetTableRows(feos.GetTableRowsRequest{
		JSON:  true,
		Scope: fmt.Sprintf("%d", name),
		Code:  "eosio.msig",
		Table: "approvals2",
		Limit: uint32(limit),
	})
	if err != nil {
		return false, nil, err
	}
	more = res.More
	info = make([]*MsigApprovalsInfo, 0)
	err = json.Unmarshal(res.Rows, &info)
	return
}

// HasRequested checks if an account is on the list of requested signatures
func (info MsigApprovalsInfo) HasRequested(actor feos.AccountName) bool {
	for _, r := range info.RequestedApprovals {
		if r.Level.Actor == actor {
			return true
		}
	}
	return info.HasApproved(actor)
}

// HasApproved checks if an account has provided a signature
func (info MsigApprovalsInfo) HasApproved(actor feos.AccountName) bool {
	for _, p := range info.ProvidedApprovals {
		if p.Level.Actor == actor {
			return true
		}
	}
	return false
}

// TODO: not sure if this is needed
type MsigExtension struct {
	Type uint16 `json:"type"`
	Data []byte `json:"data"`
}

type MsigInvalidation struct {
	Account              feos.Name      `json:"account"`
	LastInvalidationTime feos.TimePoint `json:"last_invalidation_time"`
}

type MsigOldApprovalsInfo struct {
	ProposalName       feos.Name         `json:"proposal_name"`
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
type MsigTransactionHeader feos.TransactionHeader

/*

Actions

*/

// MsigApprove approves a multi-sig proposal
type MsigApprove struct {
	Proposer     feos.AccountName `json:"proposer"`
	ProposalName feos.Name        `json:"proposal_name"`
	Level        PermissionLevel  `json:"level"`
	MaxFee       uint64           `json:"max_fee"`
	ProposalHash feos.Checksum256 `json:"proposal_hash"`
}

func NewMsigApprove(proposer feos.AccountName, proposal feos.Name, actor feos.AccountName, proposalHash feos.Checksum256) *Action {
	return NewAction("eosio.msig", "approve", actor,
		&MsigApprove{
			Proposer:     proposer,
			ProposalName: proposal,
			Level: PermissionLevel{
				Actor:      actor,
				Permission: "active",
			},
			MaxFee:       Tokens(GetMaxFee(FeeMsigApprove)),
			ProposalHash: proposalHash,
		},
	)
}

// MsigCancel withdraws a proposal, must be performed by the account that proposed the transaction
type MsigCancel struct {
	Proposer     feos.AccountName `json:"proposer"`
	ProposalName feos.Name        `json:"proposal_name"`
	Canceler     feos.AccountName `json:"canceler"`
	MaxFee       uint64           `json:"max_fee"`
}

func NewMsigCancel(proposer feos.AccountName, proposal feos.Name, actor feos.AccountName) *Action {
	return NewAction("eosio.msig", "cancel", actor,
		&MsigCancel{
			Proposer:     proposer,
			ProposalName: proposal,
			Canceler:     actor,
			MaxFee:       Tokens(GetMaxFee(FeeMsigCancel)),
		},
	)
}

// MsigExec will attempt to execute a proposed transaction
type MsigExec struct {
	Proposer     feos.AccountName `json:"proposer"`
	ProposalName feos.Name        `json:"proposal_name"`
	MaxFee       uint64           `json:"max_fee"`
	Executer     feos.AccountName `json:"executer"`
}

func NewMsigExec(proposer feos.AccountName, proposal feos.Name, fee uint64, actor feos.AccountName) *Action {
	return NewAction("eosio.msig", "exec", actor,
		&MsigExec{
			Proposer:     proposer,
			ProposalName: proposal,
			MaxFee:       fee,
			Executer:     actor,
		},
	)
}

// MsigInvalidate is used to remove all approvals and proposals for an account
type MsigInvalidate struct {
	Name   feos.Name `json:"name"`
	MaxFee uint64    `json:"max_fee"`
}

// MsigPropose is a new proposal
type MsigPropose struct {
	Proposer     feos.AccountName        `json:"proposer"`
	ProposalName feos.Name               `json:"proposal_name"`
	Requested    []*PermissionLevel      `json:"requested"`
	MaxFee       uint64                  `json:"max_fee"`
	Trx          *feos.SignedTransaction `json:"trx"`
}

type MsigWrappedPropose struct {
	Proposer     feos.AccountName   `json:"proposer"`
	ProposalName feos.Name          `json:"proposal_name"`
	Requested    []*PermissionLevel `json:"requested"`
	MaxFee       uint64             `json:"max_fee"`
	Trx          *feos.Transaction  `json:"trx"`
}

// NewMsigPropose is provided for consistency, but it will make more sense to use NewSignedMsigPropose to build *simple*
// multisig proposals since it abstracts several steps.
func NewMsigPropose(proposer feos.AccountName, proposal feos.Name, signers []*PermissionLevel, signedTx *feos.SignedTransaction) *Action {
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
func (api *API) NewSignedMsigPropose(proposalName Name, approvers []string, actions []*Action, expires time.Duration, signer *Account, txOpt *TxOptions) (*feos.PackedTransaction, error) {
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
	propTx.Expiration = feos.JSONTime{Time: time.Now().UTC().Add(expires)}
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
	newTx.Expiration = feos.JSONTime{Time: time.Now().UTC().Add(expires)}
	_, packedTx, err := api.SignTransaction(newTx, txOpt.ChainID, CompressionZlib)
	if err != nil {
		return nil, err
	}

	return packedTx, nil
}

// MsigUnapprove withdraws an existing approval for an account
type MsigUnapprove struct {
	Proposer     feos.AccountName `json:"proposer"`
	ProposalName feos.Name        `json:"proposal_name"`
	Level        PermissionLevel  `json:"level"`
	MaxFee       uint64           `json:"max_fee"`
}

func NewMsigUnapprove(proposer feos.AccountName, proposal feos.Name, actor feos.AccountName) *Action {
	return NewAction("eosio.msig", "unapprove", actor,
		&MsigUnapprove{
			Proposer:     proposer,
			ProposalName: proposal,
			Level: PermissionLevel{
				Actor:      actor,
				Permission: "active",
			},
			MaxFee: Tokens(GetMaxFee(FeeMsigUnapprove)),
		},
	)
}

type UpdateAuth struct {
	Account    feos.AccountName `json:"account"`
	Permission feos.Name        `json:"permission"`
	Parent     feos.Name        `json:"parent"`
	Auth       Authority        `json:"auth"`
	MaxFee     uint64           `json:"max_fee"`
}

// NewUpdateAuthSimple just takes a list of accounts and a threshold. Nothing fancy, most basic EOS msig account.
func NewUpdateAuthSimple(account feos.AccountName, actors []string, threshold uint32) *Action {
	acts := make([]feos.PermissionLevelWeight, 0)
	sort.Strings(actors) // actors must be sorted in ascending alphabetic order, or will get an invalid {$auth} error.
	for _, a := range actors {
		acts = append(acts, feos.PermissionLevelWeight{
			Weight:     1,
			Permission: feos.PermissionLevel{Actor: feos.AccountName(a), Permission: "active"}})
	}
	return NewAction("eosio", "updateauth", feos.AccountName(account), UpdateAuth{
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

type msigProposalRow struct {
	ProposalName      feos.Name `json:"proposal_name"`
	PackedTransaction string    `json:"packed_transaction"`
}

// MsigProposal is a query response for getting details of a proposed transaction
type MsigProposal struct {
	ProposalName      feos.Name         `json:"proposal_name"`
	PackedTransaction *feos.Transaction `json:"packed_transaction"`
	ProposalHash      feos.Checksum256  `json:"proposal_hash"`
}

// GetProposalTransaction will lookup a specific proposal
func (api *API) GetProposalTransaction(proposalAuthor feos.AccountName, proposalName feos.Name) (*MsigProposal, error) {
	name, err := feos.StringToName(string(proposalAuthor))
	if err != nil {
		return nil, err
	}
	res, err := api.GetTableRows(feos.GetTableRowsRequest{
		Code:       "eosio.msig",
		Scope:      fmt.Sprintf("%v", name),
		Table:      "proposal",
		LowerBound: string(proposalName),
		UpperBound: string(proposalName),
		Index:      "1",
		KeyType:    "name",
		Limit:      1,
		JSON:       true,
	})
	if err != nil {
		return nil, err
	}
	if len(res.Rows) < 3 {
		return nil, errors.New("did not find the proposal")
	}
	proposal := make([]*msigProposalRow, 0)
	err = json.Unmarshal(res.Rows, &proposal)
	if err != nil {
		return nil, err
	}
	txBytes, err := hex.DecodeString(proposal[0].PackedTransaction)
	decoder := feos.NewDecoder(txBytes)
	tx := &feos.Transaction{}
	err = decoder.Decode(tx)
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	h.Write(txBytes)
	sum := h.Sum(nil)
	return &MsigProposal{ProposalName: proposal[0].ProposalName, PackedTransaction: tx, ProposalHash: sum}, nil
}

type scopeResp struct {
	Scope string `json:"scope"`
	Count int    `json:"count"`
}

// GetProposals fetches the proposal list from eosio.msig returning a map of scopes, with a count for each
func (api *API) GetProposals(offset int, limit int) (more bool, scopes map[string]int, err error) {
	res, err := api.GetTableByScopeMore(feos.GetTableByScopeRequest{
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

// WrapExecute wraps a transaction to be executed with specific permissions via eosio.wrap
// NOTE: this is not working as expected, use caution.
type WrapExecute struct {
	Executor feos.AccountName  `json:"executor"`
	Trx      *feos.Transaction `json:"trx"`
}

func NewWrapExecute(actor feos.AccountName, executor feos.AccountName, trx *feos.Transaction) *Action {
	trx.Expiration = feos.JSONTime{Time: time.Unix(0, 0)}
	trx.RefBlockPrefix = 0
	trx.RefBlockNum = 0
	return NewAction("eosio.wrap", "execute", actor,
		&WrapExecute{
			Executor: executor,
			Trx:      trx,
		},
	)
}
