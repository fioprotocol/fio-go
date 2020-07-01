package fio

import (
	fos "github.com/fioprotocol/fio-go/imports/eos-fio"
)

const FioSymbol = "ᵮ"

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
	Actor          fos.AccountName `json:"actor"`
	Tpid           string          `json:"tpid"`
}

// NewTransferTokensPubKey builds an eos.Action for sending FIO tokens
func NewTransferTokensPubKey(actor fos.AccountName, recipientPubKey string, amount uint64) *Action {
	return NewAction(
		"fio.token", "trnsfiopubky", actor,
		TransferTokensPubKey{
			PayeePublicKey: recipientPubKey,
			Amount:         amount,
			MaxFee:         Tokens(GetMaxFee(FeeTransferTokensPubKey)),
			Actor:          actor,
			Tpid:           CurrentTpid(),
		},
	)
}

// Transfer is a privileged call, and not normally used for sending tokens, use TransferTokensPubKey instead
type Transfer struct {
	From     fos.AccountName `json:"from"`
	To       fos.AccountName `json:"to"`
	Quantity fos.Asset       `json:"quantity"`
	Memo     string          `json:"memo"`
}

// NewTransfer is unlikely to be called, this is a privileged action
//
// deprecated: internal action, user cannot call.
func NewTransfer(actor fos.AccountName, recipient fos.AccountName, amount uint64) *Action {
	return NewAction(
		fos.AccountName("fio.token"), "transfer", actor,
		Transfer{
			From: actor,
			To:   recipient,
			Quantity: fos.Asset{
				Amount: fos.Int64(amount),
				Symbol: fos.Symbol{
					Precision: 9,
					Symbol:    "FIO",
				},
			},
		},
	)
}

// GetBalance gets an account's balance
func (api *API) GetBalance(account fos.AccountName) (float64, error) {
	a, err := api.GetCurrencyBalance(account, "FIO", fos.AccountName("fio.token"))
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

// GetFioBalance is a convenience wrapper for GetCurrencyBalance, it is not idiomatic since it is
// not a member function of API, and will be removed in a future version
//
// deprecated: use api.GetBalance instead
func GetFioBalance(account fos.AccountName, api *API) (float64, error) {
	return api.GetBalance(account)
}
