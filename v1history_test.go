package fio

import (
	"fmt"
	"testing"
)

func TestAPI_HistGetBlockTxids(t *testing.T) {
	// fio.devtools: start.sh - option 1, 6 ... not normally started so if connect fails abort the test without failure.
	api, _, err := NewConnection(nil, "http://dev:8080")
	if err != nil {
		return
	}
	blocks, err := api.HistGetBlockTxids(123)
	if err != nil {
		t.Error(err)
		return
	}
	if len(blocks.Ids) == 0 {
		t.Error("did not get tx list")
		fmt.Println(blocks)
	}

	trace, err := api.GetTransaction(blocks.Ids[0])
	if err != nil {
		t.Error(err)
		return
	}
	if trace == nil || trace.Receipt.Status != 0 {
		t.Error("got empty tx")
	}
	fmt.Printf("%+v\n", trace)
}

func TestApi_getMaxActions(t *testing.T) {
	_, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	if !api.HasHistory() {
		fmt.Println("history api not available, skipping GetMaxActions test")
		return
	}
	h, err := api.GetMaxActions("eosio")
	if err != nil {
		t.Error(err)
		return
	}
	if h < 1000 {
		t.Errorf("eosio did not have enough action traces expected > 1000, got %d", h)
	}
}

func TestAPI_GetActionsUniq(t *testing.T) {
	_, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	if !api.HasHistory() {
		fmt.Println("history api not available, skipping GetActionsUniq test")
		return
	}
	traces, err := api.GetActionsUniq("o2ouxipw2rt4", 1000, 0)
	if err != nil {
		t.Error(err)
		return
	}
	seen := make(map[string]bool)
	for _, trace := range traces {
		if seen[trace.Receipt.ActionDigest] {
			t.Error("duplicate trace located")
			return
		}
		seen[trace.Receipt.ActionDigest] = true
	}
}
