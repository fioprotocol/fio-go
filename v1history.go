package fio

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eoscanada/eos-go"
	"io/ioutil"
	"strings"
)

type BlockTxidsResp struct {
	Ids                   []eos.Checksum256 `json:"ids"`
	LastIrreversibleBlock uint32            `json:"last_irreversible_block"`
}

func (api *API) HistGetBlockTxids(blockNum uint32) (*BlockTxidsResp, error) {
	resp, err := api.HttpClient.Post(
		api.BaseURL+"/v1/history/get_block_txids",
		"application/json",
		bytes.NewReader([]byte(fmt.Sprintf(`{"block_num": %d}`, blockNum))),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	blocks := &BlockTxidsResp{}
	err = json.Unmarshal(body, blocks)
	if err != nil {
		return nil, err
	}
	return blocks, nil
}

func (api *API) GetTransaction(id eos.Checksum256) (*eos.TransactionResp, error) {
	resp, err := api.HttpClient.Post(
		api.BaseURL+"/v1/history/get_transaction",
		"application/json",
		bytes.NewReader([]byte(fmt.Sprintf(`{"id": "%s"}`, id.String()))),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	at := &eos.TransactionResp{}
	err = json.Unmarshal(body, at)
	if err != nil {
		return nil, err
	}
	return at, nil
}

// accountActionSequence is a truncated action trace used only for finding the highest sequence number
type accountActionSequence struct {
	AccountActionSequence uint32 `json:"account_action_seq"`
}

type accountActions struct {
	Actions []accountActionSequence `json:"actions"`
}

// GetMaxActions returns the highest account_action_sequence from the get_actions endpoint.
// This is needed because paging only works with positive offsets.
func (api *API) GetMaxActions(account eos.AccountName) (highest uint32, err error) {
	resp, err := api.HttpClient.Post(
		api.BaseURL+"/v1/history/get_actions",
		"application/json",
		bytes.NewReader([]byte(fmt.Sprintf(`{"account_name":"%s","pos":-1}`, account))),
	)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	if len(body) == 0 {
		return 0, errors.New("received empty response")
	}

	aa := &accountActions{}
	err = json.Unmarshal(body, &aa)
	if err != nil {
		return 0, err
	}
	if aa.Actions == nil || len(aa.Actions) == 0 {
		return 0, nil
	}
	return aa.Actions[len(aa.Actions)-1].AccountActionSequence, nil
}

// HasHistory looks at available APIs and returns true if /v1/history/* exists.
func (api *API) HasHistory() bool {
	_, apis, err := api.GetSupportedApis()
	if err != nil {
		return false
	}
	for _, a := range apis {
		if strings.HasPrefix(a, `/v1/history/`) {
			return true
		}
	}
	return false
}
