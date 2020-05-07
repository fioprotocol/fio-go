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
}
