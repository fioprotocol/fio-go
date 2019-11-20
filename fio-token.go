package fio

import "github.com/eoscanada/eos-go"

// TransferTokensPubKey is used to send FIO tokens to a public key
type TransferTokensPubKey struct {
	PayeePublicKey string          `json:"payee_public_key"`
	Amount         uint64          `json:"amount"`
	MaxFee         uint64          `json:"max_fee"`
	Actor          eos.AccountName `json:"actor"`
	Tpid           string          `json:"tpid"`
}

// NewTransferTokensPubKey builds an eos.Action for sending FIO tokens, amount is in long form
// (9 digits, or 1000000000 = 1 FIO, see also: ConvertAmount)
func NewTransferTokensPubKey(actor eos.AccountName, recipientPubKey string, amount uint64) *eos.Action {
	return newAction(
		eos.AccountName("fio.token"), "trnsfiopubky", actor,
		TransferTokensPubKey{
			PayeePublicKey: recipientPubKey,
			Amount:         amount,
			MaxFee:         ConvertAmount(maxFees["trnsfiopubky"]),
			Actor:          actor,
			Tpid:           "",
		},
	)
}

// Transfer - unsure if this is actually used, but adding since it's in the ABI
type Transfer struct {
	From     eos.AccountName `json:"from"`
	To       eos.AccountName `json:"to"`
	Quantity eos.Asset       `json:"quantity"`
	Memo     string          `json:"memo"`
}

func NewTransfer(actor eos.AccountName, recipient eos.AccountName, amount uint64) *eos.Action {
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
		},
	)
}
