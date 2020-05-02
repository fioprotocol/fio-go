package fio

import "testing"

func TestUpdateMaxFees(t *testing.T) {
	_, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	// force the fee to be wrong, has to be done after connecting
	maxFees["add_pub_address"] = 0.0
	if ok := UpdateMaxFees(api); !ok {
		t.Error("could not update fees")
	}
	if maxFees["add_pub_address"] == 0.0 {
		t.Error("did not update")
	}
}

func TestAPI_GetFee(t *testing.T) {
	account, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	_, _, err = account.GetNames(api)
	if err != nil || len(account.Addresses) == 0 {
		t.Error("account should have an address")
		return
	}

	// tests both max fee and get fee
	max := GetMaxFee(FeeAddPubAddress)
	if max == 0 {
		t.Error("got wrong max fee")
	}
	actual, err := api.GetFee(account.Addresses[0].FioAddress, FeeAddPubAddress)
	if actual != 0 {
		t.Error("fee should have been bundled")
	}
}


