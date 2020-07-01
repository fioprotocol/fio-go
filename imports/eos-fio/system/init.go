package system

import (
	fos "github.com/fioprotocol/fio-go/imports/eos-fio"
)

func init() {
	fos.RegisterAction(AN("eosio"), ActN("setcode"), SetCode{})
	fos.RegisterAction(AN("eosio"), ActN("setabi"), SetABI{})
	fos.RegisterAction(AN("eosio"), ActN("newaccount"), NewAccount{})
	fos.RegisterAction(AN("eosio"), ActN("delegatebw"), DelegateBW{})
	fos.RegisterAction(AN("eosio"), ActN("undelegatebw"), UndelegateBW{})
	fos.RegisterAction(AN("eosio"), ActN("refund"), Refund{})
	fos.RegisterAction(AN("eosio"), ActN("regproducer"), RegProducer{})
	fos.RegisterAction(AN("eosio"), ActN("unregprod"), UnregProducer{})
	fos.RegisterAction(AN("eosio"), ActN("regproxy"), RegProxy{})
	fos.RegisterAction(AN("eosio"), ActN("voteproducer"), VoteProducer{})
	fos.RegisterAction(AN("eosio"), ActN("claimrewards"), ClaimRewards{})
	fos.RegisterAction(AN("eosio"), ActN("buyram"), BuyRAM{})
	fos.RegisterAction(AN("eosio"), ActN("buyrambytes"), BuyRAMBytes{})
	fos.RegisterAction(AN("eosio"), ActN("linkauth"), LinkAuth{})
	fos.RegisterAction(AN("eosio"), ActN("unlinkauth"), UnlinkAuth{})
	fos.RegisterAction(AN("eosio"), ActN("deleteauth"), DeleteAuth{})
	fos.RegisterAction(AN("eosio"), ActN("rmvproducer"), RemoveProducer{})
	fos.RegisterAction(AN("eosio"), ActN("setprods"), SetProds{})
	fos.RegisterAction(AN("eosio"), ActN("setpriv"), SetPriv{})
	fos.RegisterAction(AN("eosio"), ActN("canceldelay"), CancelDelay{})
	fos.RegisterAction(AN("eosio"), ActN("bidname"), Bidname{})
	// eos.RegisterAction(AN("eosio"), ActN("nonce"), &Nonce{})
	fos.RegisterAction(AN("eosio"), ActN("sellram"), SellRAM{})
	fos.RegisterAction(AN("eosio"), ActN("updateauth"), UpdateAuth{})
	fos.RegisterAction(AN("eosio"), ActN("setramrate"), SetRAMRate{})
	fos.RegisterAction(AN("eosio"), ActN("setalimits"), Setalimits{})
}

var AN = fos.AN
var PN = fos.PN
var ActN = fos.ActN
