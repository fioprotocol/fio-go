package fio

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eoscanada/eos-go"
	"sort"
	"strconv"
	"time"
)

// PermissionLevel wraps eos-go's type to add member functions
type PermissionLevel eos.PermissionLevel

func NewPermissionLevel(account eos.AccountName) *PermissionLevel {
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
		l = append(l, NewPermissionLevel(eos.AccountName(a)))
	}
	return l
}

// ToEos converts from fio.PermissionLevel to eos.PermissionLevel
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

// GetApprovals returns a list of approvals for an account
func (api *API) GetApprovals(scope Name, limit int) (more bool, info []*MsigApprovalsInfo, err error) {
	name, err := eos.StringToName(string(scope))
	if err != nil {
		return false, nil, err
	}
	res, err := api.GetTableRows(eos.GetTableRowsRequest{
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
func (info MsigApprovalsInfo) HasRequested(actor eos.AccountName) bool {
	for _, r := range info.RequestedApprovals {
		if r.Level.Actor == actor {
			return true
		}
	}
	return info.HasApproved(actor)
}

// HasApproved checks if an account has provided a signature
func (info MsigApprovalsInfo) HasApproved(actor eos.AccountName) bool {
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

// MsigApprove approves a multi-sig proposal
type MsigApprove struct {
	Proposer     eos.AccountName `json:"proposer"`
	ProposalName eos.Name        `json:"proposal_name"`
	Level        PermissionLevel `json:"level"`
	MaxFee       uint64          `json:"max_fee"`
	ProposalHash eos.Checksum256 `json:"proposal_hash"`
}

func NewMsigApprove(proposer eos.AccountName, proposal eos.Name, actor eos.AccountName, proposalHash eos.Checksum256) *Action {
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
	Proposer     eos.AccountName `json:"proposer"`
	ProposalName eos.Name        `json:"proposal_name"`
	Canceler     eos.AccountName `json:"canceler"`
	MaxFee       uint64          `json:"max_fee"`
}

func NewMsigCancel(proposer eos.AccountName, proposal eos.Name, actor eos.AccountName) *Action {
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
	Proposer     eos.AccountName `json:"proposer"`
	ProposalName eos.Name        `json:"proposal_name"`
	MaxFee       uint64          `json:"max_fee"`
	Executer     eos.AccountName `json:"executer"`
}

func NewMsigExec(proposer eos.AccountName, proposal eos.Name, fee uint64, actor eos.AccountName) *Action {
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
	Name   eos.Name `json:"name"`
	MaxFee uint64   `json:"max_fee"`
}

// MsigPropose is a new proposal
type MsigPropose struct {
	Proposer     eos.AccountName        `json:"proposer"`
	ProposalName eos.Name               `json:"proposal_name"`
	Requested    []*PermissionLevel     `json:"requested"`
	MaxFee       uint64                 `json:"max_fee"`
	Trx          *eos.SignedTransaction `json:"trx"`
}

type MsigWrappedPropose struct {
	Proposer     eos.AccountName    `json:"proposer"`
	ProposalName eos.Name           `json:"proposal_name"`
	Requested    []*PermissionLevel `json:"requested"`
	MaxFee       uint64             `json:"max_fee"`
	Trx          *eos.Transaction   `json:"trx"`
}

// NewMsigPropose is provided for consistency, but it will make more sense to use NewSignedMsigPropose to build *simple*
// multisig proposals since it abstracts several steps.
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

// MsigUnapprove withdraws an existing approval for an account
type MsigUnapprove struct {
	Proposer     eos.AccountName `json:"proposer"`
	ProposalName eos.Name        `json:"proposal_name"`
	Level        PermissionLevel `json:"level"`
	MaxFee       uint64          `json:"max_fee"`
}

func NewMsigUnapprove(proposer eos.AccountName, proposal eos.Name, actor eos.AccountName) *Action {
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

type msigProposalRow struct {
	ProposalName      eos.Name `json:"proposal_name"`
	PackedTransaction string   `json:"packed_transaction"`
}

// MsigProposal is a query response for getting details of a proposed transaction
type MsigProposal struct {
	ProposalName      eos.Name         `json:"proposal_name"`
	PackedTransaction *eos.Transaction `json:"packed_transaction"`
	ProposalHash      eos.Checksum256  `json:"proposal_hash"`
}

// GetProposalTransaction will lookup a specific proposal
func (api *API) GetProposalTransaction(proposalAuthor eos.AccountName, proposalName eos.Name) (*MsigProposal, error) {
	name, err := eos.StringToName(string(proposalAuthor))
	if err != nil {
		return nil, err
	}
	res, err := api.GetTableRows(eos.GetTableRowsRequest{
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
	decoder := eos.NewDecoder(txBytes)
	tx := &eos.Transaction{}
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

// WrapExecute wraps a transaction to be executed with specific permissions via eosio.wrap
type WrapExecute struct {
	Executor eos.AccountName  `json:"executor"`
	Trx      *eos.Transaction `json:"trx"`
}

func NewWrapExecute(actor eos.AccountName, executor eos.AccountName, trx *eos.Transaction) *Action {
	trx.Expiration = eos.JSONTime{Time: time.Unix(0, 0)}
	trx.RefBlockPrefix = 0
	trx.RefBlockNum = 0
	return NewAction("eosio.wrap", "execute", actor,
		&WrapExecute{
			Executor: executor,
			Trx:      trx,
		},
	)
}
