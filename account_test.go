package fio

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestAPI_GetFioAccount(t *testing.T) {
	api, _, _ := NewConnection(nil, "https://testnet.fio.dev")
	a, err := api.GetFioAccount("gik4jgcjciwb")
	if err != nil {
		fmt.Println(err)
	}
	j, _ := json.MarshalIndent(a, "", "  ")
	fmt.Println("\n", string(j))
}
