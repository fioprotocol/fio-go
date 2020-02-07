package fio

import (
	"errors"
	"github.com/eoscanada/eos-go"
	"sort"
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
	Time  eos.TimePoint   `json:"time"`
}

type MsigApprovalsInfo struct {
	Name               uint8          `json:"name"`
	ProposalName       eos.Name       `json:"proposal_name"`
	RequestedApprovals []MsigApproval `json:"requested_approvals"`
	ProvidedApprovals  []MsigApproval `json:"provided_approvals"`
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

// TODO: add helpers for deriving RefBlockNum and RefBlockPrefix if not already in eos-go... (these are referenced in TransactionHeader)
/*
here's a stub from a test I did previously to figure out the process ...
...
	// get current block:
	currentInfo, _ := api.GetInfo()
	// uint16: block % (2 ^ 16)
	refBlockNum := currentInfo.HeadBlockNum % uint32(math.Pow(2.0, 16.0))
	// hex -> bytes[]
	prefix, _ := hex.DecodeString(currentInfo.HeadBlockID.String())
	// take last 24 bytes to fit, convert to uint32 (little endian)
	refBlockPrefix := binary.LittleEndian.Uint32(prefix[8:])
	fmt.Println("expecting ref_block_num of: ", refBlockNum)
	fmt.Printf("expecting ref_block_prefix of: %d\n\n", refBlockPrefix)
    // build a new tx:
	transferPub := fio.NewTransferTokensPubKey(account.Actor, "FIO5wuXscTZrb65e9WmdZN2G2hyxtZ3SA1mr6edz9G217x9CySbME", fio.Tokens(100.0))
	tx := fio.NewTransaction([]*fio.Action{transferPub}, opts)
	// print it out
	j, _ := json.MarshalIndent(tx, "", "  ")
	fmt.Println(string(j))
...
*/

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
		MaxFee:       Tokens(GetMaxFee(FeeMsigPropose))*feeBytes,
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
			MaxFee:       Tokens(GetMaxFee(FeeMsigPropose))*feeBytes,
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
