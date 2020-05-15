package fio

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eoscanada/eos-go"
	"io/ioutil"
	"sort"
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

// GetActionsUniq strips the results of GetActions of duplicate traces, this can occur with certain transactions
// that may have multiple actors involved and the same trace is presented more than once but associated with a
// different actor. This will give preference to the trace referencing the actor queried if possible.
func (api *API) GetActionsUniq(actor eos.AccountName, offset int64, pos int64) ([]*eos.ActionTrace, error) {
	traceUniq := make(map[string]*eos.ActionTrace)
	resp, err := api.GetActions(eos.GetActionsRequest{AccountName:actor, Offset: eos.Int64(offset), Pos: eos.Int64(pos)})
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Actions) == 0 {
		return nil, errors.New("empty result")
	}
	for i := range resp.Actions {
		// use a closure to dereference
		func (act *eos.ActionResp) {
			// have we already seen this act_digest?
			switch traceUniq[act.Trace.Receipt.ActionDigest] {
			case nil:
				traceUniq[act.Trace.Receipt.ActionDigest] = &act.Trace
			default:
				// if there is a dup, prefer the correct actor, otherwise just ignore and keep the existing:
				if act.Trace.Receipt.AuthSequence != nil && len(act.Trace.Receipt.AuthSequence) > 0 && act.Trace.Receipt.AuthSequence[0].Account == actor {
					traceUniq[act.Trace.Receipt.ActionDigest] = &act.Trace
				}
			}
		}(&resp.Actions[i])
	}
	traces := make([]*eos.ActionTrace, 0)
	for _, v := range traceUniq {
		traces = append(traces, v)
	}
	sort.Slice(traces, func(i, j int) bool {
		return traces[i].Receipt.GlobalSequence < traces[j].Receipt.GlobalSequence
	})
	return traces, nil
}