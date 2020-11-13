package eos

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
)

/*
These are FIO-specific modifications to eos-go
 */

// checkFeeRange is a safety mechanism to check if an action has a fee and prevents under/over flows.
// Not all fees are consistently one type, some are uint64 and some are int64.
// All of the structures in fio-go treat them as a uint64 for consistency.
func checkFeeRange(action *Action) error {
	switch true {
	case action == nil:
		return errors.New("validateFee: invalid, Action is nil")
	case action.HexData != nil && action.Data == nil:
		// only check if an embedded struct exists
		return nil
	case action.Data == nil:
		return errors.New("checkFeeRange: invalid, Data is nil")
	case reflect.TypeOf(action.ActionData.Data).Kind() != reflect.Struct:
		return errors.New("checkFeeRange: invalid, Data is not a struct")
	}

	maxFee := reflect.ValueOf(action.ActionData.Data).FieldByName("MaxFee")
	if !maxFee.IsValid() {
		// MaxFee doesn't exist, all clear
		return nil
	}
	switch maxFee.Kind() {
	case reflect.Uint64:
		return checkUnderOver(maxFee.Uint())
	case reflect.Int64:
		return checkUnderOver(maxFee.Uint())
	case reflect.Float32, reflect.Float64:
		return errors.New("checkFeeRange cannot be a float")
	case reflect.String:
		i, err := strconv.ParseInt(maxFee.String(), 10, 64)
		if err != nil {
			return err
		}
		return checkUnderOver(i)
	}

	return errors.New(fmt.Sprintf("checkFeeRange: cannot validate type (%s) for MaxFee, allowed types are uint64, int64, and string", maxFee.Kind().String()))
}

// checkUnderOver throws an error if an int64 < 0 or uint64 > 9,223,372,036,854,775,807 to prevent sending out of range
// values to nodeos which will allow over/under flows.
func checkUnderOver(v interface{}) error {
	switch v.(type) {
	case uint64:
		if v.(uint64) > math.MaxInt64 {
			return errors.New("checkUnderOver: fee could overflow int64")
		}
		return nil
	case int64:
		if v.(int64) < 0 {
			return errors.New("checkUnderOver: fee could underflow uint64")
		}
		return nil
	}
	return errors.New("checkUnderOver: not an int64 or uint64")
}

