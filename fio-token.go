package fio

import (
	"encoding/json"
	"github.com/eoscanada/eos-go"
	"sync"
)

const (
	FeeRegisterFioAddress   = "register_fio_address"
	FeeAddPubAddress        = "add_pub_address"
	FeeRegisterFioDomain    = "register_fio_domain"
	FeeRenewFioDomain       = "renew_fio_domain"
	FeeRenewFioAddress      = "renew_fio_address"
	FeeBurnExpired          = "burnexpired"
	FeeSetDomainPub         = "setdomainpub"
	FeeTransferTokensPubKey = "transfer_tokens_pub_key"
	FeeRecordSend           = "record_send"
	FeeNewFundsRequest      = "new_funds_request"
	FeeRejectFundsRequest   = "reject_funds_request"
)

var (
	// maxFees holds the fees for transactions
	// use fio.GetMaxFee() instead of directly accessing this map to ensure concurrent safe access
	// IMPORTANT: these are default values: call `fio.UpdateMaxFees` to refresh values from the on-chain table.
	maxFees = map[string]float64{
		"register_fio_address":        5.0,
		"add_pub_address":             0.01,
		"register_fio_domain":         40.0,
		"renew_fio_domain":            40.0,
		"renew_fio_address":           5.0,
		"burnexpired":                 0.1,
		"setdomainpub":                0.1,
		"transfer_tokens_fio_address": 0.1,
		"transfer_tokens_pub_key":     0.25,
		"record_send":                 0.1,
		"new_funds_request":           0.1,
		"reject_funds_request":        0.1,
	}
	maxFeeMutex    = sync.RWMutex{}
	maxFeesUpdated = false
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

// Tokens is a convenience function for converting from a float for human readability.
// Example 1 FIO Token: Tokens(1.0) == uint64(1000000000)
func Tokens(tokens float64) uint64 {
	return uint64(tokens * 1000000000.0)
}

// TransferTokensPubKey is used to send FIO tokens to a public key
type TransferTokensPubKey struct {
	PayeePublicKey string          `json:"payee_public_key"`
	Amount         uint64          `json:"amount"`
	MaxFee         uint64          `json:"max_fee"`
	Actor          eos.AccountName `json:"actor"`
	Tpid           string          `json:"tpid"`
}

// NewTransferTokensPubKey builds an eos.Action for sending FIO tokens
func NewTransferTokensPubKey(actor eos.AccountName, recipientPubKey string, amount uint64) *Action {
	return newAction(
		eos.AccountName("fio.token"), "trnsfiopubky", actor,
		TransferTokensPubKey{
			PayeePublicKey: recipientPubKey,
			Amount:         amount,
			MaxFee:         Tokens(GetMaxFee("transfer_tokens_pub_key")),
			Actor:          actor,
			Tpid:           globalTpid,
		},
	)
}

// Transfer - unsure if this is actually used, but adding since it's in the ABI
type Transfer struct {
	From     eos.AccountName `json:"from"`
	To       eos.AccountName `json:"to"`
	Quantity eos.Asset       `json:"quantity"`
	Memo     string          `json:"memo"`
	MaxFee   uint64          `json:"max_fee"`
}

func NewTransfer(actor eos.AccountName, recipient eos.AccountName, amount uint64) *Action {
	return newAction(
		eos.AccountName("fio.token"), "transfer", actor,
		Transfer{
			From: actor,
			To:   recipient,
			Quantity: eos.Asset{
				Amount: eos.Int64(amount),
				Symbol: eos.Symbol{
					Precision: 9,
					Symbol:    "FIO",
				},
			},
			MaxFee: Tokens(GetMaxFee("transfer_tokens_fio_address")),
		},
	)
}

func GetFioBalance(account eos.AccountName, api *API) (float64, error) {
	a, err := api.GetCurrencyBalance(account, "FIO", eos.AccountName("fio.token"))
	if err != nil {
		return 0.0, err
	}
	if len(a) > 0 {
		if a[0].Amount > 0 {
			return float64(a[0].Amount) / 1000000000.0, nil
		}
	}
	return 0.0, nil
}
