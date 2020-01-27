package fio

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

// API struct allows extending the eos.API with FIO-specific functions
type API struct {
	eos.API
}

// Action struct duplicates eos.Action
type Action struct {
	Account       eos.AccountName       `json:"account"`
	Name          eos.ActionName        `json:"name"`
	Authorization []eos.PermissionLevel `json:"authorization,omitempty"`
	eos.ActionData
}

type TxOptions struct {
	eos.TxOptions
}

func (txo TxOptions) toEos() *eos.TxOptions {
	return &eos.TxOptions{
		ChainID:          txo.ChainID,
		HeadBlockID:      txo.HeadBlockID,
		MaxNetUsageWords: txo.MaxNetUsageWords,
		DelaySecs:        txo.DelaySecs,
		MaxCPUUsageMS:    txo.MaxCPUUsageMS,
		Compress:         txo.Compress,
	}
}

// copy over CompressionTypes to reduce need for eos-go imports
const (
	CompressionNone = eos.CompressionType(iota)
	CompressionZlib
)

// NewTransaction wraps eos.NewTransaction
func NewTransaction(actions []*Action, txOpts *TxOptions) *eos.Transaction {
	eosActions := make([]*eos.Action, 0)
	for _, a := range actions {
		eosActions = append(
			eosActions,
			&eos.Action{
				Account:       a.Account,
				Name:          a.Name,
				Authorization: a.Authorization,
				ActionData:    a.ActionData,
			},
		)
	}
	return eos.NewTransaction(eosActions, txOpts.toEos())
}

// NewConnection sets up the API interface for interacting with the FIO API
func NewConnection(keyBag *eos.KeyBag, url string) (*API, *TxOptions, error) {
	var api = eos.New(url)
	api.SetSigner(keyBag)
	api.SetCustomGetRequiredKeys(
		func(tx *eos.Transaction) (keys []ecc.PublicKey, e error) {
			return keyBag.AvailableKeys()
		},
	)
	txOpts := &TxOptions{}
	err := txOpts.FillFromChain(api)
	if err != nil {
		return &API{}, nil, err
	}
	a := &API{*api}
	if !maxFeesUpdated {
		_ = UpdateMaxFees(a)
	}
	return a, txOpts, nil
}

// NewAction creates an Action for FIO contract calls
func NewAction(contract eos.AccountName, name eos.ActionName, actor eos.AccountName, actionData interface{}) *Action {
	return &Action{
		Account: contract,
		Name:    name,
		Authorization: []eos.PermissionLevel{
			{
				Actor:      actor,
				Permission: "active",
			},
		},
		ActionData: eos.NewActionData(actionData),
	}
}

// GetCurrentBlock provides the current head block number
func (api API) GetCurrentBlock() (blockNum uint32) {
	info, err := api.GetInfo()
	if err != nil {
		return
	}
	return info.HeadBlockNum
}

// WaitForConfirm checks if a tx made it on-chain, it uses brute force to search a range of
// blocks since the eos.GetTransaction doesn't provide a block hint argument, it will continue
// to search for roughly 30 seconds and then timeout. If there is an error it sets the returned block
// number to 1
func (api API) WaitForConfirm(firstBlock uint32, txid string) (block uint32, err error) {
	if txid == "" {
		return 1, errors.New("invalid txid")
	}
	var loopErr error
	tested := make(map[uint32]bool)
	for i := 0; i < 30; i++ {
		// allow at least one block to be produced before searching
		time.Sleep(time.Second)
		latest := api.GetCurrentBlock()
		if firstBlock == 0 || firstBlock > latest {
			return 1, errors.New("invalid starting block provided")
		}
		if latest == uint32(1<<32-1) {
			continue
		}
		// note, this purposely doesn't check the head block until next run since that occasionally
		// results in a false-negative
		for b := firstBlock; firstBlock < latest; b++ {
			if !tested[b] {
				blockResp, err := api.GetBlockByNum(b)
				if err != nil {
					loopErr = err
					time.Sleep(time.Second)
					continue
				}
				tested[b] = true
				for _, trx := range blockResp.SignedBlock.Transactions {
					if trx.Transaction.ID.String() == txid {
						return b, nil
					}
				}
			}
		}
	}
	if loopErr != nil {
		return 1, loopErr
	}
	return 1, errors.New("timeout waiting for confirmation")
}

// WaitForPreCommit waits until 180 blocks (minimum pre-commit limit) have passed given a block number.
// It makes sense to set seconds to the same value (180).
func (api API) WaitForPreCommit(block uint32, seconds int) (err error) {
	if block == 0 || block == 1<<32-1 {
		return errors.New("invalid block")
	}
	for i := 0; i < seconds; i++ {
		info, err := api.GetInfo()
		if err != nil {
			return err
		}
		if info.HeadBlockNum >= block+180 {
			return nil
		}
		time.Sleep(time.Second)
	}
	return errors.New("timeout waiting for minimum pre-committed block")
}

// WaitForIrreversible waits until the most recent irreversible block is greater than the specified block.
// Normally this will be about 360 seconds.
func (api API) WaitForIrreversible(block uint32, seconds int) (err error) {
	if block == 0 || block == 1<<32-1 {
		return errors.New("invalid block")
	}
	for i := 0; i < seconds; i++ {
		info, err := api.GetInfo()
		if err != nil {
			return err
		}
		if info.LastIrreversibleBlockNum >= block {
			return nil
		}
		time.Sleep(time.Second)
	}
	return errors.New("timeout waiting for irreversible block")
}

// PushEndpointRaw is adapted from eos-go call() function in api.go to allow overriding the endpoint for a push-transaction
// the endpoint provided should be the full path to the endpoint such as "/v1/chain/push_transaction"
func (api API) PushEndpointRaw(endpoint string, body interface{}) (out json.RawMessage, err error) {
	enc := func(v interface{}) (io.Reader, error) {
		if v == nil {
			return nil, nil
		}
		buffer := &bytes.Buffer{}
		encoder := json.NewEncoder(buffer)
		encoder.SetEscapeHTML(false)
		err := encoder.Encode(v)
		if err != nil {
			return nil, err
		}
		return buffer, nil
	}
	jsonBody, err := enc(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", api.BaseURL+endpoint, jsonBody)
	if err != nil {
		return nil, fmt.Errorf("NewRequest: %s", err)
	}
	for k, v := range api.Header {
		if req.Header == nil {
			req.Header = http.Header{}
		}
		req.Header[k] = append(req.Header[k], v...)
	}
	resp, err := api.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", req.URL.String(), err)
	}
	defer resp.Body.Close()
	var cnt bytes.Buffer
	_, err = io.Copy(&cnt, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Copy: %s", err)
	}
	if resp.StatusCode == 404 {
		var apiErr eos.APIError
		if err := json.Unmarshal(cnt.Bytes(), &apiErr); err != nil {
			return nil, eos.ErrNotFound
		}
		return nil, apiErr
	}
	if resp.StatusCode > 299 {
		var apiErr eos.APIError
		if err := json.Unmarshal(cnt.Bytes(), &apiErr); err != nil {
			return nil, fmt.Errorf("%s: status code=%d, body=%s", req.URL.String(), resp.StatusCode, cnt.String())
		}
		// Handle cases where some API calls (/v1/chain/get_account for example) returns a 500
		// error when retrieving data that does not exist.
		if apiErr.IsUnknownKeyError() {
			return nil, eos.ErrNotFound
		}
		return nil, apiErr
	}
	if err := json.Unmarshal(cnt.Bytes(), &out); err != nil {
		return nil, fmt.Errorf("Unmarshal: %s", err)
	}
	return out, nil
}

type Producers struct {
	Producers               []Producer `json:"producers"`
	TotalProducerVoteWeight string     `json:"total_producer_vote_weight"`
	More                    string     `json:"more"`
}

type Producer struct {
	Owner         eos.AccountName `json:"owner"`
	FioAddress    Address         `json:"fio_address"`
	TotalVotes    string          `json:"total_votes"`
	IsActive      uint8           `json:"is_active"`
	Url           string          `json:"url"`
	UnpaidBlocks  uint64          `json:"unpaid_blocks"`
	LastClaimTime string          `json:"last_claim_time"`
	Location      uint8           `json:"location"`
}

// The producers table is a litte different on FIO, use this instead of the GetProducers call from eos-go:
func (api API) GetFioProducers() (fioProducers *Producers, err error) {
	req, err := http.NewRequest("POST", api.BaseURL+`/v1/chain/get_producers`, nil)
	if err != nil {
		return &Producers{}, err
	}
	req.Header.Add("content-type", "application/json")
	res, err := api.HttpClient.Do(req)
	if err != nil {
		return &Producers{}, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return &Producers{}, err
	}
	err = json.Unmarshal(body, &fioProducers)
	if err != nil {
		return &Producers{}, err
	}
	return
}

// AllABIs returns a map of every ABI available. This is only possible in FIO because there are a small number
// of contracts that exist.
func (api API) AllABIs() (map[eos.AccountName]*eos.ABI, error) {
	type contracts struct {
		Owner string `json:"owner"`
	}
	table, err := api.GetTableRows(eos.GetTableRowsRequest{
		Code:  "eosio",
		Scope: "eosio",
		Table: "abihash",
		JSON:  true,
	})
	if err != nil {
		return nil, err
	}
	result := make([]contracts, 0)
	_ = json.Unmarshal(table.Rows, &result)
	abiList := make(map[eos.AccountName]*eos.ABI)
	for _, name := range result {
		bi, err := api.GetABI(eos.AccountName(name.Owner))
		if err != nil {
			continue
		}
		abiList[bi.AccountName] = &bi.ABI
	}
	if len(abiList) == 0 {
		return nil, errors.New("could not get abis from eosio tables")
	}
	return abiList, nil
}
