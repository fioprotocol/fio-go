package fio

import (
	"github.com/eoscanada/eos-go"
)

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
	return NewAction(
		eos.AccountName("fio.token"), "trnsfiopubky", actor,
		TransferTokensPubKey{
			PayeePublicKey: recipientPubKey,
			Amount:         amount,
			MaxFee:         Tokens(GetMaxFee(FeeTransferTokensPubKey)),
			Actor:          actor,
			Tpid:           CurrentTpid(),
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
	return NewAction(
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
			MaxFee: Tokens(GetMaxFee(FeeTransferTokensPubKey)),
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
