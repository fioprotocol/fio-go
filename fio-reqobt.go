package fio

type ObtContent struct {
	PayerPublicAddress string `json:"payer_public_address"`
	PayeePublicAddress string `json:"payee_public_address"`
	Amount             string `json:"amount"`
	TokenCode          string `json:"token_code"`
	Status             string `json:"status"`
	ObtId              string `json:"obt_id"`
	Memo               string `json:"memo"`
	Hash               string `json:"hash"`
	OfflineUrl         string `json:"offline_url"`
}

// TODO: need to figure out how to encrypt the Content field, no use building further until that works ...
type RecordSend struct {
	FioRequestId    string `json:"fio_request_id"`
	PayerFioAddress string `json:"payer_fio_address"`
	PayeeFioAddress string `json:"payee_fio_address"`
	Content         string `json:"content"`
	MaxFee          uint64 `json:"max_fee"`
	Actor           string `json:"actor"` // NOTE this differs from other fio.* contracts, and is a string not name!!!
	Tpid            string `json:"tpid"`
}

type NewFundsReq struct {
	PayerFioAddress string `json:"payer_fio_address"`
	PayeeFioAddress string `json:"payee_fio_address"`
	Content         string `json:"content"`
	MaxFee          uint64 `json:"max_fee"`
	Actor           string `json:"actor"`
	Tpid            string `json:"tpid"`
}

type RejectFndReq struct {
	FioRequestId string `json:"fio_request_id"`
	MaxFee       uint64 `json:"max_fee"`
	Actor        string `json:"actor"`
	Tpid         string `json:"tpid"`
}
