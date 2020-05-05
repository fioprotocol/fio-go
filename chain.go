package fio

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eoscanada/eos-go"
	eosecc "github.com/eoscanada/eos-go/ecc"
	"github.com/fioprotocol/fio-go/eos-go/ecc"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
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

// TxOptions wraps eos.TxOptions
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
		func(tx *eos.Transaction) (keys []eosecc.PublicKey, e error) {
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

// NewAction creates an Action for FIO contract calls, assumes the permission is "active"
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
//
// deprecated: use NewActionWithPermission instead
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

// NewActionWithPermission allows building an action and specifying the permission
func NewActionWithPermission(contract eos.AccountName, name eos.ActionName, actor eos.AccountName, permission string, actionData interface{}) *Action {
	return &Action{
		Account: contract,
		Name:    name,
		Authorization: []eos.PermissionLevel{
			{
				Actor:      actor,
				Permission: eos.PermissionName(permission),
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

// GetTableRowsOrderRequest extends eos.GetTableRowsRequest by adding a reverse field for sorting on index, not sure
// if it is something unique to FIO or missing for eos-go, but is very handy for limiting searches.
type GetTableRowsOrderRequest struct {
	Code       string `json:"code"` // Contract "code" account where table lives
	Scope      string `json:"scope"`
	Table      string `json:"table"`
	LowerBound string `json:"lower_bound,omitempty"`
	UpperBound string `json:"upper_bound,omitempty"`
	Limit      uint32 `json:"limit,omitempty"`          // defaults to 10 => chain_plugin.hpp:struct get_table_rows_params
	KeyType    string `json:"key_type,omitempty"`       // The key type of --index, primary only supports (i64), all others support (i64, i128, i256, float64, float128, ripemd160, sha256). Special type 'name' indicates an account name.
	Index      string `json:"index_position,omitempty"` // Index number, 1 - primary (first), 2 - secondary index (in order defined by multi_index), 3 - third index, etc. Number or name of index can be specified, e.g. 'secondary' or '2'.
	EncodeType string `json:"encode_type,omitempty"`    // The encoding type of key_type (i64 , i128 , float64, float128) only support decimal encoding e.g. 'dec'" "i256 - supports both 'dec' and 'hex', ripemd160 and sha256 is 'hex' only
	JSON       bool   `json:"json"`                     // JSON output if true, binary if false
	Reverse    bool   `json:"reverse"`                  // Sort order
}

// GetTableRowsOrder duplicates eos.GetTableRows but adds a Reverse flag
func (api *API) GetTableRowsOrder(gtro GetTableRowsOrderRequest) (*eos.GetTableRowsResp, error) {
	j, err := json.Marshal(&gtro)
	if err != nil {
		return nil, err
	}
	resp, err := api.HttpClient.Post(api.BaseURL+"/v1/chain/get_table_rows", "application/json", bytes.NewReader(j))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	tableRows := &eos.GetTableRowsResp{}
	err = json.Unmarshal(body, tableRows)
	if err != nil {
		return nil, err
	}
	return tableRows, nil
}

// GetRefBlockFor calculates the Reference for an arbitrary block and ID
func GetRefBlockFor(blocknum uint32, id string) (refBlockNum uint32, refBlockPrefix uint32, err error) {
	// uint16: block % (2 ^ 16)
	refBlockNum = blocknum % uint32(math.Pow(2.0, 16.0))
	prefix, err := hex.DecodeString(id)
	if err != nil {
		return 0, 0, err
	}
	// take last 24 bytes to fit, convert to uint32 (little endian)
	refBlockPrefix = binary.LittleEndian.Uint32(prefix[8:])
	return
}

// GetRefBlock calculates a the block reference for the last irreversible block
func (api *API) GetRefBlock() (refBlockNum uint32, refBlockPrefix uint32, err error) {
	// get current block:
	currentInfo, err := api.GetInfo()
	if err != nil {
		return 0, 0, err
	}
	// uint16: block % (2 ^ 16)
	return GetRefBlockFor(currentInfo.LastIrreversibleBlockNum, currentInfo.LastIrreversibleBlockID.String())
}

type BlockrootMerkle struct {
	ActiveNodes []eos.Checksum256 `json:"_active_nodes"`
	NodeCount   uint32            `json:"_node_count"`
}

type protocolFeatures struct {
	ProtocolFeatures []interface{} `json:"protocol_features"` // not sure what goes here, leaving private
}

// BlockHeader duplicates eos.BlockHeader to allow using the modified ecc package
type BlockHeader struct {
	Timestamp        eos.BlockTimestamp `json:"timestamp"`
	Producer         eos.AccountName    `json:"producer"`
	Confirmed        uint16             `json:"confirmed"`
	Previous         eos.Checksum256    `json:"previous"`
	TransactionMRoot eos.Checksum256    `json:"transaction_mroot"`
	ActionMRoot      eos.Checksum256    `json:"action_mroot"`
	ScheduleVersion  uint32             `json:"schedule_version"`
	NewProducers     *Schedule          `json:"new_producers" eos:"optional"`
	HeaderExtensions []*eos.Extension   `json:"header_extensions"`
}

// BlockHeaderState holds information about reversible blocks.
type BlockHeaderState struct {
	BlockNum                  uint32            `json:"block_num"`
	ProposedIrrBlock          uint32            `json:"dpos_proposed_irreversible_blocknum"`
	IrrBlock                  uint32            `json:"dpos_irreversible_blocknum"`
	ActiveSchedule            *Schedule         `json:"active_schedule"`
	BlockrootMerkle           BlockrootMerkle   `json:"blockroot_merkle"`
	ProducerToLastProduced    []json.RawMessage `json:"producer_to_last_produced"` // array of arrays with mixed types, access via member func
	ProducerToLastImpliedIrb  []json.RawMessage `json:"producer_to_last_implied_irb"`
	BlockSigningKey           ecc.PublicKey     `json:"block_signing_key"`
	ConfirmCount              []int             `json:"confirm_count"`
	Id                        eos.Checksum256   `json:"id"`
	Header                    *BlockHeader      `json:"header"`
	PendingSchedule           *Schedule         `json:"pending_schedule"`
	ActivatedProtocolFeatures protocolFeatures  `json:"activated_protocol_features"`
}

type BlockHeaderStateReq struct {
	BlockNumOrId interface{} `json:"block_num_or_id"` // can be checksum or uint32
}

// GetBlockHeaderState returns the details for a reversible block. If the block is irreversible the api will return an error.
func (api *API) GetBlockHeaderState(numOrId interface{}) (*BlockHeaderState, error) {
	reqJson, err := json.Marshal(&BlockHeaderStateReq{BlockNumOrId: numOrId})
	if err != nil {
		return nil, err
	}
	resp, err := api.HttpClient.Post(api.BaseURL+"/v1/chain/get_block_header_state", "application/json", bytes.NewReader(reqJson))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, errors.New("get_block_header_state: empty reply")
	}
	bhs := &BlockHeaderState{}
	err = json.Unmarshal(body, bhs)
	if err != nil {
		return nil, err
	}
	return bhs, nil
}

const (
	ProducerToLastProduced uint8 = iota
	ProducerToLastImplied
)

type ProducerToLast struct {
	Producer          eos.AccountName `json:"producer"`
	BlockNum          uint32          `json:"block_num"`
	ProducedOrImplied string          `json:"produced_or_implied"`
}

// ProducerToLast extracts a slice of ProducerToLast structs from a BlockHeaderState, this contains either the last
// block that the producer signed, or the last irreversible block. This is useful for seeing if a producer is missing
// rounds, or is responsible for double-signed blocks causing forks.
func (bhs *BlockHeaderState) ProducerToLast(producedOrImplied uint8) (found bool, last []*ProducerToLast) {
	var l []json.RawMessage
	var pOrI string
	switch producedOrImplied {
	case ProducerToLastProduced:
		if bhs.ProducerToLastProduced == nil || len(bhs.ProducerToLastProduced) == 0 {
			return false, nil
		}
		l = bhs.ProducerToLastProduced
		pOrI = "producer_to_last_produced"
	case ProducerToLastImplied:
		if bhs.ProducerToLastImpliedIrb == nil || len(bhs.ProducerToLastImpliedIrb) == 0 {
			return false, nil
		}
		l = bhs.ProducerToLastImpliedIrb
		pOrI = "producer_to_last_implied_irb"
	}
	last = make([]*ProducerToLast, 0)
	for _, ptl := range l {
		pl := &ProducerToLast{}
		iToPtl := make([]interface{}, 0)
		err := json.Unmarshal(ptl, &iToPtl)
		if err != nil {
			continue
		}
		for _, v := range iToPtl {
			switch v.(type) {
			case string:
				pl.Producer = eos.AccountName(v.(string))
				continue
			case float64:
				pl.BlockNum = uint32(v.(float64))
			}
		}
		if pl.BlockNum != 0 {
			pl.ProducedOrImplied = pOrI
			last = append(last, pl)
		}
	}
	if len(last) > 0 {
		found = true
	}
	return
}
