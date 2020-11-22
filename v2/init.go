package fio

import "github.com/blockpane/eos-go/ecc"

func init() {
	ecc.SetPublicKeyPrefixCompat("FIO")
}
