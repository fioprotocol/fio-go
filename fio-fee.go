package fio

import (
	"encoding/json"
	"github.com/eoscanada/eos-go"
	"sync"
)

const (
	FeeAddPubAddress        = "add_pub_address"
	FeeAddToWhitelist       = "add_to_whitelist"
	FeeAuthDelete           = "auth_delete"
	FeeAuthLink             = "auth_link"
	FeeAuthUpdate           = "auth_update"
	FeeBurnExpired          = "burnexpired"
	FeeMsigApprove          = "msig_approve"
	FeeMsigCancel           = "msig_cancel"
	FeeMsigExec             = "msig_exec"
	FeeMsigInvalidate       = "msig_invalidate"
	FeeMsigPropose          = "msig_propose"
	FeeMsigUnapprove        = "msig_unapprove"
	FeeNewFundsRequest      = "new_funds_request"
	FeeProxyVote            = "proxy_vote"
	FeeRecordObtData        = "record_obt_data"
	FeeRecordSend           = "record_send" // outdated endpoint name
	FeeRegisterFioAddress   = "register_fio_address"
	FeeRegisterFioDomain    = "register_fio_domain"
	FeeRegisterProducer     = "register_producer"
	FeeRegisterProxy        = "register_proxy"
	FeeRejectFundsRequest   = "reject_funds_request"
	FeeRemoveFromWhitelist  = "remove_from_whitelist"
	FeeRenewFioAddress      = "renew_fio_address"
	FeeRenewFioDomain       = "renew_fio_domain"
	FeeSetDomainPub         = "set_fio_domain_public"
	FeeSubmitBundledTrans   = "submit_bundled_transaction"
	FeeTransferTokensPubKey = "transfer_tokens_pub_key"
	FeeUnregisterProducer   = "unregister_producer"
	FeeUnregisterProxy      = "unregister_proxy"
	FeeVoteProducer         = "vote_producer"
)

var (
	// maxFees holds the fees for transactions
	// use fio.GetMaxFee() instead of directly accessing this map to ensure concurrent safe access
	// IMPORTANT: these are default values: call `fio.UpdateMaxFees` to refresh values from the on-chain table.
	maxFees = map[string]float64{
		"add_pub_address":             0.4,
		"add_to_whitelist":            0.0,
		"auth_delete":                 0.4,
		"auth_link":                   0.4,
		"auth_update":                 0.4,
		"burnexpired":                 0.1,
		"msig_approve":                0.4,
		"msig_cancel":                 0.4,
		"msig_exec":                   0.4,
		"msig_invalidate":             0.4,
		"msig_propose":                0.4,
		"msig_unapprove":              0.4,
		"new_funds_request":           0.8,
		"proxy_vote":                  0.4,
		"record_obt_data":             0.8,
		"record_send":                 0.8, // outdated endpoint name.
		"register_fio_address":        40.0,
		"register_fio_domain":         800.0,
		"register_producer":           200.0,
		"register_proxy":              0.4,
		"reject_funds_request":        0.4,
		"remove_from_whitelist":       0.0,
		"renew_fio_address":           40.0,
		"renew_fio_domain":            800.0,
		"set_fio_domain_public":       0.4,
		"setdomainpub":                0.4, // outdated endpoint name.
		"submit_bundled_transaction":  0.0,
		"transfer_tokens_fio_address": 0.1,
		"transfer_tokens_pub_key":     2.0,
		"unregister_proxy":            0.4,
		"vote_producer":               0.4,
	}
	// maxFeesByAction correlates fee name to action name, useful when working directly with contracts, not API endpoint
	// slight chance fee will be wrong if there are two actions with identical name, but don't think there are any cases
	// where that will happen right now.
	maxFeesByAction = map[string]string{
		"deleteauth":   FeeAuthDelete,
		"linkauth":     FeeAuthLink,
		"regproducer":  FeeRegisterProducer,
		"regproxy":     FeeRegisterProxy,
		"unregprod":    FeeUnregisterProducer,
		"unregproxy":   FeeUnregisterProxy,
		"updateauth":   FeeAuthUpdate,
		"voteproducer": FeeVoteProducer,
		"voteproxy":    FeeProxyVote,
		"approve":      FeeMsigApprove,
		"cancel":       FeeMsigCancel,
		"exec":         FeeMsigExec,
		"invalidate":   FeeMsigInvalidate,
		"propose":      FeeMsigPropose,
		"unapprove":    FeeMsigUnapprove,
		"addaddress":   FeeAddPubAddress,
		"regaddress":   FeeRegisterFioAddress,
		"regdomain":    FeeRegisterFioDomain,
		"renewaddress": FeeRenewFioAddress,
		"renewdomain":  FeeRenewFioDomain,
		"setdomainpub": FeeSetDomainPub,
		"newfundsreq":  FeeNewFundsRequest,
		"recordobt":    FeeRecordObtData,
		"rejectfndreq": FeeRejectFundsRequest,
		"trnsfiopubky": FeeTransferTokensPubKey,
	}
	maxFeeActionMutex = sync.RWMutex{}
	maxFeeMutex       = sync.RWMutex{}
	maxFeesUpdated    = false
)

// UpdateMaxFees refreshes the maxFees map from the on-chain table. This is automatically called
// by NewConnection if fees are not already up-to-date.
func UpdateMaxFees(api *API) bool {
	type feeRow struct {
		EndPoint  string `json:"end_point"`
		SufAmount uint64 `json:"suf_amount"`
	}
	fees, err := api.GetTableRows(eos.GetTableRowsRequest{
		Code:  "fio.fee",
		Scope: "fio.fee",
		Table: "fiofees",
		Limit: 100,
		JSON:  true,
	})
	if err != nil {
		return false
	}
	results := make([]feeRow, 0)
	err = json.Unmarshal(fees.Rows, &results)
	if err != nil {
		return false
	}
	maxFeeMutex.Lock()
	for _, f := range results {
		maxFees[f.EndPoint] = float64(f.SufAmount) / 1000000000.0
	}
	maxFeeMutex.Unlock()
	maxFeesUpdated = true
	return true
}

// GetMaxFee looks up a fee from the map
func GetMaxFee(name string) (fioTokens float64) {
	maxFeeMutex.RLock()
	fioTokens = maxFees[name]
	maxFeeMutex.RUnlock()
	return fioTokens
}

func GetMaxFeeByAction(name string) (fioTokens float64) {
	maxFeeMutex.RLock()
	maxFeeActionMutex.RLock()
	fioTokens = maxFees[maxFeesByAction[name]]
	maxFeeMutex.RUnlock()
	maxFeeActionMutex.RUnlock()
	return fioTokens
}

// MaxFeesUpdated checks if the fee map has been updated, or if using the default (possibly wrong) values
func MaxFeesUpdated() bool {
	return maxFeesUpdated
}

// MaxFeesJson provides a JSON representation of the current fee map
func MaxFeesJson() []byte {
	maxFeeMutex.RLock()
	j, _ := json.MarshalIndent(maxFees, "", "  ")
	maxFeeMutex.RUnlock()
	return j
}

type CreateFee struct {
	EndPoint  string `json:"end_point"`
	Type      uint64 `json:"type"`
	SufAmount uint64 `json:"suf_amount"`
}

type FeeValue struct {
	EndPoint string `json:"end_point"`
	Value    uint64 `json:"value"`
}

type SetFeeVote struct {
	FeeRatios []FeeValue `json:"fee_ratios"`
	Actor     string     `json:"actor"`
}

func NewSetFeeVote(ratios []FeeValue, actor eos.AccountName) *Action {
	return NewAction("fio.fee", "setfeevote", actor,
		SetFeeVote{
			FeeRatios: ratios,
			Actor:     string(actor),
		})
}

type BundleVote struct {
	BundledTransactions uint64 `json:"bundled_transactions"`
	Actor               string `json:"actor"`
}

func NewBundleVote(transactions uint64, actor eos.AccountName) *Action {
	return NewAction("eosio", "bundlevote", actor,
		BundleVote{
			BundledTransactions: transactions,
			Actor:               string(actor),
		},
	)
}

type SetFeeMult struct {
	Multiplier float64 `json:"multiplier"`
	Actor      string  `json:"actor"`
}

type FioFee struct {
	FeeId        uint64      `json:"fee_id"`
	EndPoint     string      `json:"end_point"`
	EndPointHash eos.Uint128 `json:"end_point_hash"`
	Type         uint64      `json:"type"`
	SufAmount    uint64      `json:"suf_amount"`
}

type FeeVoter struct {
	BlockProducerName eos.AccountName `json:"block_producer_name"`
	FeeMultiplier     float64         `json:"fee_multiplier"`
	LastVoteTimestamp uint64          `json:"lastvotetimestamp"`
}

type FeeVote struct {
	Id                uint64          `json:"id"`
	BlockProducerName eos.AccountName `json:"block_producer_name"`
	EndPoint          string          `json:"end_point"`
	EndPointHash      uint64          `json:"end_point_hash"`
	SufAmount         uint64          `json:"suf_amount"`
	LastVoteTimestamp uint64          `json:"lastvotetimestamp"`
}

type BundleVoter struct {
	BlockProducerName eos.AccountName `json:"block_producer_name"`
	BundleVoteNumber  uint64          `json:"bundlevotenumber"`
	LastVoteTimestamp uint64          `json:"lastvotetimestamp"`
}
