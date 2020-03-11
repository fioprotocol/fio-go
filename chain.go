package fio

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
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

func (act Action) ToEos() *eos.Action {
	return &eos.Action{
		Account:       act.Account,
		Name:          act.Name,
		Authorization: act.Authorization,
		ActionData:    act.ActionData,
	}
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
	api.Header.Set("User-Agent", "fio-go")
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

// NewActionAsOwner is the same as NewAction, but specifies the owner permission, really only needed for msig updateauth in FIO
func NewActionAsOwner(contract eos.AccountName, name eos.ActionName, actor eos.AccountName, actionData interface{}) *Action {
	return &Action{
		Account: contract,
		Name:    name,
		Authorization: []eos.PermissionLevel{
			{
				Actor:      actor,
				Permission: "owner",
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
	return api.WaitMaxForConfirm(firstBlock, txid, 30)
}

func (api API) WaitMaxForConfirm(firstBlock uint32, txid string, seconds int) (block uint32, err error) {
	if txid == "" {
		return 1, errors.New("invalid txid")
	}
	if seconds <= 1 {
		return 1, errors.New("must wait at least 2 seconds")
	}
	// allow at least one block to be produced before searching
	time.Sleep(time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(seconds)*time.Second))
	defer cancel()
	tick := time.NewTicker(time.Second)
	tested := make(map[uint32]bool)
	for {
		select {
		case <-tick.C:
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
						return 1, err
					}
					tested[b] = true
					for _, trx := range blockResp.SignedBlock.Transactions {
						if trx.Transaction.ID.String() == txid {
							return b, nil
						}
					}
				}
			}
		case <-ctx.Done():
			return 1, errors.New("timeout waiting for confirmation")
		}
	}
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

// used to deal with string vs bool in More field:
type getTableByScopeResp struct {
	More interface{}     `json:"more"`
	Rows json.RawMessage `json:"rows"`
}

// GetTableByScopeMore handles responses that have either a bool or a string as the more response.
func (api API) GetTableByScopeMore(request eos.GetTableByScopeRequest) (*eos.GetTableByScopeResp, error) {
	reqBody, err := json.Marshal(&request)
	if err != nil {
		return nil, err
	}
	resp, err := api.HttpClient.Post(api.BaseURL+"/v1/chain/get_table_by_scope", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	gt := &getTableByScopeResp{}
	err = json.Unmarshal(body, gt)
	if err != nil {
		return nil, err
	}
	var more bool
	moreString, ok := gt.More.(string)
	if ok {
		more, err = strconv.ParseBool(moreString)
		if err != nil && moreString != "" {
			more = true // if it's not empty, we have more.
		}
	} else {
		moreBool, valid := gt.More.(bool)
		if valid {
			more = moreBool
		}
	}
	return &eos.GetTableByScopeResp{
		More: more,
		Rows: gt.Rows,
	}, nil
}

func (api *API) GetRefBlock() (refBlockNum uint32, refBlockPrefix uint32, err error) {
	// get current block:
	currentInfo, err := api.GetInfo()
	if err != nil {
		return 0, 0, err
	}
	// uint16: block % (2 ^ 16)
	refBlockNum = currentInfo.HeadBlockNum % uint32(math.Pow(2.0, 16.0))
	prefix, err := hex.DecodeString(currentInfo.HeadBlockID.String())
	if err != nil {
		return 0, 0, err
	}
	// take last 24 bytes to fit, convert to uint32 (little endian)
	refBlockPrefix = binary.LittleEndian.Uint32(prefix[8:])
	return
}

