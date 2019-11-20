package fio

import (
	"github.com/eoscanada/eos-go"
	"sync"
)


// use GetMaxFee() instead of directly accessing this map to ensure concurrent safe access
var (
	maxFees = map[string]float64{
		"regaddress":   5.0,
		"addaddress":   1.0,
		"regdomain":    50.0,
		"renewdomain":  50.0,
		"renewaddress": 5.0,
		"burnexpired":  0.3,
		"setdomainpub": 0.3,
		"transfer":     0.3,
		"trnsfiopubky": 0.3,
		"recordsend":   0.3,
		"newfundsreq":  0.3,
		"rejectfndreq": 0.3,
	}
	maxFeeMutex = sync.RWMutex{}
)

func GetMaxFee(name string) (fee float64) {
	maxFeeMutex.RLock()
	fee = maxFees[name]
	maxFeeMutex.RUnlock()
	return fee
}

// ConvertAmount is a convenience function for converting from a float for human readability
func ConvertAmount(tokens float64) uint64 {
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

// NewTransferTokensPubKey builds an eos.Action for sending FIO tokens, amount is in long form
// (9 digits, or 1000000000 = 1 FIO, see also: ConvertAmount)
func NewTransferTokensPubKey(actor eos.AccountName, recipientPubKey string, amount uint64) *eos.Action {
	return newAction(
		eos.AccountName("fio.token"), "trnsfiopubky", actor,
		TransferTokensPubKey{
			PayeePublicKey: recipientPubKey,
			Amount:         amount,
			MaxFee:         ConvertAmount(GetMaxFee("trnsfiopubky")),
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
