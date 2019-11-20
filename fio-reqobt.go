package fio

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
