package fio

import (
	"errors"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	"time"
)

// NewConnection sets up the eos.API interface for interacting with the FIO API
func NewConnection(keyBag *eos.KeyBag, url string) (*eos.API, *eos.TxOptions, error) {
	api := eos.New(url)
	api.SetSigner(keyBag)
	api.SetCustomGetRequiredKeys(
		func(tx *eos.Transaction) (keys []ecc.PublicKey, e error) {
			return keyBag.AvailableKeys()
		},
	)
	txOpts := &eos.TxOptions{}
	if err := txOpts.FillFromChain(api); err != nil {
		return nil, nil, err
	}
	return api, txOpts, nil
}

// newAction creates an eos.Action for FIO contract calls
func newAction(contract eos.AccountName, name eos.ActionName, actor eos.AccountName, actionData interface{}) *eos.Action {
	return &eos.Action{
		Account:       contract,
		Name:          name,
		Authorization: []eos.PermissionLevel{
			{
				Actor:      actor,
				Permission: "active",
			},
		},
		ActionData:    eos.NewActionData(actionData),
	}
}

// GetCurrentBlock provides the current head block number
func GetCurrentBlock(api *eos.API) (blockNum uint32) {
	info, err := api.GetInfo()
	if err != nil {
		blockNum = 1<<32-1
		return
	}
	return info.HeadBlockNum
}

// WaitForConfirm checks if a tx made it on-chain, it uses brute force to search a range of
// blocks since the eos.GetTransaction doesn't provide a block hint argument, it will continue
// to search for roughly 30 seconds and then timeout. If there is an error it sets the returned block
// number to the upper limit of a uint32
func WaitForConfirm(firstBlock uint32, txid string, api *eos.API) (block uint32, err error) {
	if txid == "" {
		return 1<<32-1, errors.New("invalid txid")
	}
	// allow at least one block to be produced before searching
	time.Sleep(time.Second)
	var loopErr error
	tested := make(map[uint32]bool)
	for i := 0; i < 30; i++ {
		latest := GetCurrentBlock(api)
		if firstBlock == 0 || firstBlock > latest {
			return 1<<32-1, errors.New("invalid starting block provided")
		}
		if latest == uint32(1<<32-1){
			time.Sleep(time.Second)
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
		return 1<<32-1, loopErr
	}
	return 1<<32-1, errors.New("timeout waiting for confirmation")
}

// WaitForPreCommit waits until 180 blocks (minimum pre-commit limit) have passed given a block number.
// It makes sense to set seconds to the same value (180).
func WaitForPreCommit(block uint32, seconds int, api *eos.API) (err error) {
	if block == 0 || block == 1<<32-1 {
		return errors.New("invalid block")
	}
	for i := 0; i < seconds; i++ {
		info, err := api.GetInfo()
		if err != nil {
			return err
		}
		if info.HeadBlockNum >= block + 180 {
			return nil
		}
		time.Sleep(time.Second)
	}
	return errors.New("timeout waiting for minimum pre-committed block")
}

// WaitForIrreversible waits until the most recent irreversible block is greater than the specified block.
// Normally this will be about 360 seconds.
func WaitForIrreversible(block uint32, seconds int, api *eos.API) (err error) {
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

/*
TODO: use reflection to allow setting the Tpid in an Action if the field exists:
// Action is a clone of eos.Action so it can have custom member functions
type Action eos.Action

func (a *Action) SetTpid(tpid string) error {
	actionType := reflect.TypeOf(a.ActionData.Data)
	value, ok := actionType.FieldByName(`Tpid`)
	if !ok {
		return errors.New("transaction does not contain a tpid field")
	}
	reflect.ValueOf(value).Set("tpid")
}

 */