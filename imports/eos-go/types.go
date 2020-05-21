package eos

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/eoscanada/eos-go/ecc"
)

var symbolRegex = regexp.MustCompile("^[0-9],[A-Z]{1,7}$")
var symbolCodeRegex = regexp.MustCompile("^[A-Z]{1,7}$")

// For reference:
// https://github.com/mithrilcoin-io/EosCommander/blob/master/app/src/main/java/io/mithrilcoin/eoscommander/data/remote/model/types/EosByteWriter.java

type Name string
type AccountName Name
type PermissionName Name
type ActionName Name
type TableName Name
type ScopeName Name

func AN(in string) AccountName    { return AccountName(in) }
func ActN(in string) ActionName   { return ActionName(in) }
func PN(in string) PermissionName { return PermissionName(in) }

type AccountResourceLimit struct {
	Used      Int64 `json:"used"`
	Available Int64 `json:"available"`
	Max       Int64 `json:"max"`
}

type DelegatedBandwidth struct {
	From      AccountName `json:"from"`
	To        AccountName `json:"to"`
	NetWeight Asset       `json:"net_weight"`
	CPUWeight Asset       `json:"cpu_weight"`
}

type TotalResources struct {
	Owner     AccountName `json:"owner"`
	NetWeight Asset       `json:"net_weight"`
	CPUWeight Asset       `json:"cpu_weight"`
	RAMBytes  Int64       `json:"ram_bytes"`
}

type VoterInfo struct {
	Owner             AccountName   `json:"owner"`
	Proxy             AccountName   `json:"proxy"`
	Producers         []AccountName `json:"producers"`
	Staked            Int64         `json:"staked"`
	LastVoteWeight    JSONFloat64   `json:"last_vote_weight"`
	ProxiedVoteWeight JSONFloat64   `json:"proxied_vote_weight"`
	IsProxy           byte          `json:"is_proxy"`
}

type RefundRequest struct {
	Owner       AccountName `json:"owner"`
	RequestTime JSONTime    `json:"request_time"` //         {"name":"request_time", "type":"time_point_sec"},
	NetAmount   Asset       `json:"net_amount"`
	CPUAmount   Asset       `json:"cpu_amount"`
}

type CompressionType uint8

const (
	CompressionNone = CompressionType(iota)
	CompressionZlib
)

func (c CompressionType) String() string {
	switch c {
	case CompressionNone:
		return "none"
	case CompressionZlib:
		return "zlib"
	default:
		return ""
	}
}

func (c CompressionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

func (c *CompressionType) UnmarshalJSON(data []byte) error {
	tryNext, err := c.tryUnmarshalJSONAsString(data)
	if err != nil && !tryNext {
		return err
	}

	if tryNext {
		_, err := c.tryUnmarshalJSONAsUint8(data)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *CompressionType) tryUnmarshalJSONAsString(data []byte) (tryNext bool, err error) {
	var s string
	err = json.Unmarshal(data, &s)
	if err != nil {
		_, isTypeError := err.(*json.UnmarshalTypeError)

		// Let's continue with next handler is we hit a type error, might be an integer...
		return isTypeError, err
	}

	switch s {
	case "none":
		*c = CompressionNone
	case "zlib":
		*c = CompressionZlib
	default:
		return false, fmt.Errorf("unknown compression type %s", s)
	}

	return false, nil
}

func (c *CompressionType) tryUnmarshalJSONAsUint8(data []byte) (tryNext bool, err error) {
	var s uint8
	err = json.Unmarshal(data, &s)
	if err != nil {
		return false, err
	}

	switch s {
	case 0:
		*c = CompressionNone
	case 1:
		*c = CompressionZlib
	default:
		return false, fmt.Errorf("unknown compression type %d", s)
	}

	return false, nil
}

// CurrencyName

type CurrencyName string

type Bool bool

func (b *Bool) UnmarshalJSON(data []byte) error {
	var num int
	err := json.Unmarshal(data, &num)
	if err == nil {
		*b = Bool(num != 0)
		return nil
	}

	var boolVal bool
	if err := json.Unmarshal(data, &boolVal); err != nil {
		return fmt.Errorf("couldn't unmarshal bool as int or true/false: %s", err)
	}

	*b = Bool(boolVal)
	return nil
}

// Asset

// NOTE: there's also ExtendedAsset which is a quantity with the attached contract (AccountName)
type Asset struct {
	Amount Int64
	Symbol
}

func (a Asset) Add(other Asset) Asset {
	if a.Symbol != other.Symbol {
		panic("Add applies only to assets with the same symbol")
	}
	return Asset{Amount: a.Amount + other.Amount, Symbol: a.Symbol}
}

func (a Asset) Sub(other Asset) Asset {
	if a.Symbol != other.Symbol {
		panic("Sub applies only to assets with the same symbol")
	}
	return Asset{Amount: a.Amount - other.Amount, Symbol: a.Symbol}
}

func (a Asset) String() string {
	amt := a.Amount
	if amt < 0 {
		amt = -amt
	}
	strInt := fmt.Sprintf("%d", amt)
	if len(strInt) < int(a.Symbol.Precision+1) {
		// prepend `0` for the difference:
		strInt = strings.Repeat("0", int(a.Symbol.Precision+uint8(1))-len(strInt)) + strInt
	}

	var result string
	if a.Symbol.Precision == 0 {
		result = strInt
	} else {
		result = strInt[:len(strInt)-int(a.Symbol.Precision)] + "." + strInt[len(strInt)-int(a.Symbol.Precision):]
	}
	if a.Amount < 0 {
		result = "-" + result
	}

	return fmt.Sprintf("%s %s", result, a.Symbol.Symbol)
}

type ExtendedAsset struct {
	Asset    Asset `json:"asset"`
	Contract AccountName
}

// NOTE: there's also a new ExtendedSymbol (which includes the contract (as AccountName) on which it is)
type Symbol struct {
	Precision uint8
	Symbol    string

	// Caching of symbol code if it was computed once
	symbolCode uint64
}

func NameToSymbol(name Name) (Symbol, error) {
	symbol := Symbol{}
	value, err := StringToName(string(name))
	if err != nil {
		return symbol, fmt.Errorf("name %s is invalid: %s", name, err)
	}

	symbol.Precision = uint8(value & 0xFF)
	symbol.Symbol = SymbolCode(value >> 8).String()

	return symbol, nil
}

func StringToSymbol(str string) (Symbol, error) {
	symbol := Symbol{}
	if !symbolRegex.MatchString(str) {
		return symbol, fmt.Errorf("%s is not a valid symbol", str)
	}

	precision, _ := strconv.ParseUint(string(str[0]), 10, 8)

	symbol.Precision = uint8(precision)
	symbol.Symbol = str[2:]

	return symbol, nil
}

func MustStringToSymbol(str string) Symbol {
	symbol, err := StringToSymbol(str)
	if err != nil {
		panic("invalid symbol " + str)
	}

	return symbol
}

func (s Symbol) SymbolCode() (SymbolCode, error) {
	if s.symbolCode != 0 {
		return SymbolCode(s.symbolCode), nil
	}

	symbolCode, err := StringToSymbolCode(s.Symbol)
	if err != nil {
		return 0, err
	}

	return SymbolCode(symbolCode), nil
}

func (s Symbol) MustSymbolCode() SymbolCode {
	symbolCode, err := StringToSymbolCode(s.Symbol)
	if err != nil {
		panic("invalid symbol code " + s.Symbol)
	}

	return symbolCode
}

func (s Symbol) ToUint64() (uint64, error) {
	symbolCode, err := s.SymbolCode()
	if err != nil {
		return 0, fmt.Errorf("symbol %s is not a valid symbol code: %s", s.Symbol, err)
	}

	return uint64(symbolCode)<<8 | uint64(s.Precision), nil
}

func (s Symbol) ToName() (string, error) {
	u, err := s.ToUint64()
	if err != nil {
		return "", err
	}
	return NameToString(u), nil
}

func (s Symbol) String() string {
	return fmt.Sprintf("%d,%s", s.Precision, s.Symbol)
}

type SymbolCode uint64

func NameToSymbolCode(name Name) (SymbolCode, error) {
	value, err := StringToName(string(name))
	if err != nil {
		return 0, fmt.Errorf("name %s is invalid: %s", name, err)
	}

	return SymbolCode(value), nil
}

func StringToSymbolCode(str string) (SymbolCode, error) {
	if len(str) > 7 {
		return 0, fmt.Errorf("string is too long to be a valid symbol_code")
	}

	var symbolCode uint64
	for i := len(str) - 1; i >= 0; i-- {
		if str[i] < 'A' || str[i] > 'Z' {
			return 0, fmt.Errorf("only uppercase letters allowed in symbol_code string")
		}

		symbolCode <<= 8
		symbolCode = symbolCode | uint64(str[i])
	}

	return SymbolCode(symbolCode), nil
}

func (sc SymbolCode) ToName() string {
	return NameToString(uint64(sc))
}

func (sc SymbolCode) String() string {
	builder := strings.Builder{}

	symbolCode := uint64(sc)
	for i := 0; i < 7; i++ {
		if symbolCode == 0 {
			return builder.String()
		}

		builder.WriteByte(byte(symbolCode & 0xFF))
		symbolCode >>= 8
	}

	return builder.String()
}

// EOSSymbol represents the standard EOS symbol on the chain.  It's
// here just to speed up things.
var EOSSymbol = Symbol{Precision: 4, Symbol: "EOS"}

// REXSymbol represents the standard REX symbol on the chain.  It's
// here just to speed up things.
var REXSymbol = Symbol{Precision: 4, Symbol: "REX"}

func NewEOSAsset(amount int64) Asset {
	return Asset{Amount: Int64(amount), Symbol: EOSSymbol}
}

// NewAsset reads from a string an EOS asset.
//
// Deprecated: Use `NewAssetFromString` instead
func NewAsset(in string) (out Asset, err error) {
	return NewAssetFromString(in)
}

// NewAssetFromString reads a string an decode it to an eos.Asset
// structure if possible. The input must contains an amount and
// a symbol. The precision is inferred based on the actual number
// of decimals present.
func NewAssetFromString(in string) (out Asset, err error) {
	out, err = newAssetFromString(in)
	if err != nil {
		return out, err
	}

	if out.Symbol.Symbol == "" {
		return out, fmt.Errorf("invalid format %q, expected an amount and a currency symbol", in)
	}

	return
}

func NewEOSAssetFromString(input string) (Asset, error) {
	return NewFixedSymbolAssetFromString(EOSSymbol, input)
}

func NewREXAssetFromString(input string) (Asset, error) {
	return NewFixedSymbolAssetFromString(REXSymbol, input)
}

func NewFixedSymbolAssetFromString(symbol Symbol, input string) (out Asset, err error) {
	integralPart, decimalPart, symbolPart, err := splitAsset(input)
	if err != nil {
		return out, err
	}

	symbolCode := symbol.MustSymbolCode().String()
	precision := symbol.Precision

	if len(decimalPart) > int(precision) {
		return out, fmt.Errorf("symbol %s precision mismatch: expected %d, got %d", symbol, precision, len(decimalPart))
	}

	if symbolPart != "" && symbolPart != symbolCode {
		return out, fmt.Errorf("symbol %s code mismatch: expected %s, got %s", symbol, symbolCode, symbolPart)
	}

	if len(decimalPart) < int(precision) {
		decimalPart += strings.Repeat("0", int(precision)-len(decimalPart))
	}

	val, err := strconv.ParseInt(integralPart+decimalPart, 10, 64)
	if err != nil {
		return out, err
	}

	return Asset{
		Amount: Int64(val),
		Symbol: Symbol{Precision: precision, Symbol: symbolCode},
	}, nil
}

func newAssetFromString(in string) (out Asset, err error) {
	integralPart, decimalPart, symbolPart, err := splitAsset(in)
	if err != nil {
		return out, err
	}

	val, err := strconv.ParseInt(integralPart+decimalPart, 10, 64)
	if err != nil {
		return out, err
	}

	out.Amount = Int64(val)
	out.Symbol.Precision = uint8(len(decimalPart))
	out.Symbol.Symbol = symbolPart

	return
}

func splitAsset(input string) (integralPart, decimalPart, symbolPart string, err error) {
	input = strings.Trim(input, " ")
	if len(input) == 0 {
		return "", "", "", fmt.Errorf("input cannot be empty")
	}

	parts := strings.Split(input, " ")
	if len(parts) >= 1 {
		integralPart, decimalPart, err = splitAssetAmount(parts[0])
		if err != nil {
			return
		}
	}

	if len(parts) == 2 {
		symbolPart = parts[1]
		if len(symbolPart) > 7 {
			return "", "", "", fmt.Errorf("invalid asset %q, symbol should have less than 7 characters", input)
		}
	}

	if len(parts) > 2 {
		return "", "", "", fmt.Errorf("invalid asset %q, expecting an amount alone or an amount and a currency symbol", input)
	}

	return
}

func splitAssetAmount(input string) (integralPart, decimalPart string, err error) {
	parts := strings.Split(input, ".")
	switch len(parts) {
	case 1:
		integralPart = parts[0]
	case 2:
		integralPart = parts[0]
		decimalPart = parts[1]

		if len(decimalPart) > math.MaxUint8 {
			err = fmt.Errorf("invalid asset amount precision %q, should have less than %d characters", input, math.MaxUint8)

		}
	default:
		return "", "", fmt.Errorf("invalid asset amount %q, expected amount to have at most a single dot", input)
	}

	return
}

func (a *Asset) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}

	asset, err := NewAsset(s)
	if err != nil {
		return err
	}

	*a = asset

	return nil
}

func (a Asset) MarshalJSON() (data []byte, err error) {
	return json.Marshal(a.String())
}

type Permission struct {
	PermName     string    `json:"perm_name"`
	Parent       string    `json:"parent"`
	RequiredAuth Authority `json:"required_auth"`
}

type PermissionLevel struct {
	Actor      AccountName    `json:"actor"`
	Permission PermissionName `json:"permission"`
}

// NewPermissionLevel parses strings like `account@active`,
// `otheraccount@owner` and builds a PermissionLevel struct. It
// validates that there is a single optional @ (where permission
// defaults to 'active'), and validates length of account and
// permission names.
func NewPermissionLevel(in string) (out PermissionLevel, err error) {
	parts := strings.Split(in, "@")
	if len(parts) > 2 {
		return out, fmt.Errorf("permission %q invalid, use account[@permission]", in)
	}

	if len(parts[0]) > 12 {
		return out, fmt.Errorf("account name %q too long", parts[0])
	}

	out.Actor = AccountName(parts[0])
	out.Permission = PermissionName("active")
	if len(parts) == 2 {
		if len(parts[1]) > 12 {
			return out, fmt.Errorf("permission %q name too long", parts[1])
		}

		out.Permission = PermissionName(parts[1])
	}

	return
}

type PermissionLevelWeight struct {
	Permission PermissionLevel `json:"permission"`
	Weight     uint16          `json:"weight"` // weight_type
}

type Authority struct {
	Threshold uint32                  `json:"threshold"`
	Keys      []KeyWeight             `json:"keys,omitempty"`
	Accounts  []PermissionLevelWeight `json:"accounts,omitempty"`
	Waits     []WaitWeight            `json:"waits,omitempty"`
}

type KeyWeight struct {
	PublicKey ecc.PublicKey `json:"key"`
	Weight    uint16        `json:"weight"` // weight_type
}

type WaitWeight struct {
	WaitSec uint32 `json:"wait_sec"`
	Weight  uint16 `json:"weight"` // weight_type
}

type GetRawCodeAndABIResp struct {
	AccountName  AccountName `json:"account_name"`
	WASMasBase64 string      `json:"wasm"`
	ABIasBase64  string      `json:"abi"`
}

type GetCodeResp struct {
	AccountName AccountName `json:"account_name"`
	CodeHash    string      `json:"code_hash"`
	WASM        string      `json:"wasm"`
	ABI         ABI         `json:"abi"`
}

type GetCodeHashResp struct {
	AccountName AccountName `json:"account_name"`
	CodeHash    string      `json:"code_hash"`
}

type GetABIResp struct {
	AccountName AccountName `json:"account_name"`
	ABI         ABI         `json:"abi"`
}

type ABIJSONToBinResp struct {
	Binargs string `json:"binargs"`
}

type ABIBinToJSONResp struct {
	Args M `json:"args"`
}

// JSONTime

type JSONTime struct {
	time.Time
}

const JSONTimeFormat = "2006-01-02T15:04:05"

func (t JSONTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", t.Format(JSONTimeFormat))), nil
}

func (t *JSONTime) UnmarshalJSON(data []byte) (err error) {
	if string(data) == "null" {
		return nil
	}

	t.Time, err = time.Parse(`"`+JSONTimeFormat+`"`, string(data))
	return err
}

// ParseJSONTime will parse a string into a JSONTime object
func ParseJSONTime(date string) (JSONTime, error) {
	var t JSONTime
	var err error
	t.Time, err = time.Parse(JSONTimeFormat, string(date))
	return t, err
}

// HexBytes

type HexBytes []byte

func (t HexBytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(t))
}

func (t *HexBytes) UnmarshalJSON(data []byte) (err error) {
	var s string
	err = json.Unmarshal(data, &s)
	if err != nil {
		return
	}

	*t, err = hex.DecodeString(s)
	return
}

func (t HexBytes) String() string {
	return hex.EncodeToString(t)
}

// Checksum256

type Checksum160 []byte

func (t Checksum160) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(t))
}
func (t *Checksum160) UnmarshalJSON(data []byte) (err error) {
	var s string
	err = json.Unmarshal(data, &s)
	if err != nil {
		return
	}

	*t, err = hex.DecodeString(s)
	return
}

type Checksum256 []byte

func (t Checksum256) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(t))
}
func (t *Checksum256) UnmarshalJSON(data []byte) (err error) {
	var s string
	err = json.Unmarshal(data, &s)
	if err != nil {
		return
	}

	*t, err = hex.DecodeString(s)
	return
}

func (t Checksum256) String() string {
	return hex.EncodeToString(t)
}

type Checksum512 []byte

func (t Checksum512) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(t))
}
func (t *Checksum512) UnmarshalJSON(data []byte) (err error) {
	var s string
	err = json.Unmarshal(data, &s)
	if err != nil {
		return
	}

	*t, err = hex.DecodeString(s)
	return
}

// SHA256Bytes is deprecated and renamed to Checksum256 for
// consistency. Please update your code as this type will eventually
// be phased out.
type SHA256Bytes = Checksum256

type Varuint32 uint32
type Varint32 int32

// Tstamp

type Tstamp struct {
	time.Time
}

func (t Tstamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%d", t.UnixNano()))
}

func (t *Tstamp) UnmarshalJSON(data []byte) (err error) {
	var unixNano int64
	if data[0] == '"' {
		var s string
		if err = json.Unmarshal(data, &s); err != nil {
			return
		}

		unixNano, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}

	} else {
		unixNano, err = strconv.ParseInt(string(data), 10, 64)
		if err != nil {
			return err
		}
	}

	*t = Tstamp{time.Unix(0, unixNano)}

	return nil
}

// BlockNum extracts the block number (or height) from a hex-encoded block ID.
func BlockNum(blockID string) uint32 {
	if len(blockID) < 8 {
		return 0
	}
	bin, err := hex.DecodeString(blockID[:8])
	if err != nil {
		return 0
	}
	return binary.BigEndian.Uint32(bin)
}

type BlockTimestamp struct {
	time.Time
}

const BlockTimestampFormat = "2006-01-02T15:04:05.999"

func (t BlockTimestamp) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", t.Format(BlockTimestampFormat))), nil
}

func (t *BlockTimestamp) UnmarshalJSON(data []byte) (err error) {
	if string(data) == "null" {
		return nil
	}

	t.Time, err = time.Parse(`"`+BlockTimestampFormat+`"`, string(data))
	if err != nil {
		t.Time, err = time.Parse(`"`+BlockTimestampFormat+`Z07:00"`, string(data))
	}
	return err
}

// TimePoint represents the number of microseconds since EPOCH (Jan 1st 1970)
type TimePoint uint64

// TimePointSec represents the number of seconds since EPOCH (Jan 1st 1970)
type TimePointSec uint32

type JSONFloat64 float64

func (f *JSONFloat64) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return errors.New("empty value")
	}

	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}

		val, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}

		*f = JSONFloat64(val)

		return nil
	}

	var fl float64
	if err := json.Unmarshal(data, &fl); err != nil {
		return err
	}

	*f = JSONFloat64(fl)

	return nil
}

// JSONInt64 is deprecated in favor of Int64.
type JSONInt64 = Int64

type Int64 int64

func (i Int64) MarshalJSON() (data []byte, err error) {
	if i > 0xffffffff || i < -0xffffffff {
		encodedInt, err := json.Marshal(int64(i))
		if err != nil {
			return nil, err
		}
		data = append([]byte{'"'}, encodedInt...)
		data = append(data, '"')
		return data, nil
	}
	return json.Marshal(int64(i))
}

func (i *Int64) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return errors.New("empty value")
	}

	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}

		val, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}

		*i = Int64(val)

		return nil
	}

	var v int64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*i = Int64(v)

	return nil
}

type Uint64 uint64

func (i Uint64) MarshalJSON() (data []byte, err error) {
	if i > 0xffffffff {
		encodedInt, err := json.Marshal(uint64(i))
		if err != nil {
			return nil, err
		}
		data = append([]byte{'"'}, encodedInt...)
		data = append(data, '"')
		return data, nil
	}
	return json.Marshal(uint64(i))
}

func (i *Uint64) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return errors.New("empty value")
	}

	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}

		val, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}

		*i = Uint64(val)

		return nil
	}

	var v uint64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*i = Uint64(v)

	return nil
}

type Uint128 struct {
	Lo uint64
	Hi uint64
}

type Int128 Uint128

type Float128 Uint128

// func (i Int128) BigInt() *big.Int {
// 	// decode the Lo and Hi to handle the sign
// 	return nil
// }

// func (i Uint128) BigInt() *big.Int {
// 	// no sign to handle, all good..
// 	return nil
// }

// func NewInt128(i *big.Int) (Int128, error) {
// 	// if the big Int overflows the JSONInt128 limits..
// 	return Int128{}, nil
// }

// func NewUint128(i *big.Int) (Uint128, error) {
// 	// if the big Int overflows the JSONInt128 limits..
// 	return Uint128{}, nil
// }

func (i Uint128) MarshalJSON() (data []byte, err error) {
	return json.Marshal(i.String())
}

func (i Int128) MarshalJSON() (data []byte, err error) {
	return json.Marshal(Uint128(i).String())
}

func (i Float128) MarshalJSON() (data []byte, err error) {
	return json.Marshal(Uint128(i).String())
}

func (i Uint128) String() string {
	// Same for Int128, Float128
	number := make([]byte, 16)
	binary.LittleEndian.PutUint64(number[:], i.Lo)
	binary.LittleEndian.PutUint64(number[8:], i.Hi)
	return fmt.Sprintf("0x%s%s", hex.EncodeToString(number[:8]), hex.EncodeToString(number[8:]))
}

func (i *Int128) UnmarshalJSON(data []byte) error {
	var el Uint128
	if err := json.Unmarshal(data, &el); err != nil {
		return err
	}

	out := Int128(el)
	*i = out

	return nil
}

func (i *Float128) UnmarshalJSON(data []byte) error {
	var el Uint128
	if err := json.Unmarshal(data, &el); err != nil {
		return err
	}

	out := Float128(el)
	*i = out

	return nil
}

func (i *Uint128) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	if !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		return fmt.Errorf("int128 expects 0x prefix")
	}

	truncatedVal := s[2:]
	if len(truncatedVal) != 32 {
		return fmt.Errorf("int128 expects 32 characters after 0x, had %d", len(truncatedVal))
	}

	loHex := truncatedVal[:16]
	hiHex := truncatedVal[16:]

	lo, err := hex.DecodeString(loHex)
	if err != nil {
		return err
	}

	hi, err := hex.DecodeString(hiHex)
	if err != nil {
		return err
	}

	loUint := binary.LittleEndian.Uint64(lo)
	hiUint := binary.LittleEndian.Uint64(hi)

	i.Lo = loUint
	i.Hi = hiUint

	return nil
}

// Blob

// Blob is base64 encoded data
// https://github.com/EOSIO/fc/blob/0e74738e938c2fe0f36c5238dbc549665ddaef82/include/fc/variant.hpp#L47
type Blob string

// Data returns decoded base64 data
func (b Blob) Data() ([]byte, error) {
	return base64.StdEncoding.DecodeString(string(b))
}

// String returns the blob as a string
func (b Blob) String() string {
	return string(b)
}
