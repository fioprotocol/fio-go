package fio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/eoscanada/eos-go"
	"io/ioutil"
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
