package fio

import (
	"encoding/json"
	"fmt"
	"github.com/fioprotocol/fio-go/eos"
	"regexp"
)

// BurnNfts is intended to be called by block producers to remove expired NFT mappings from RAM
type BurnNfts struct{}

func NewBurnNfts(actor eos.AccountName) *Action {
	return NewAction(
		"fio.address", "burnnfts", actor,
		BurnNfts{},
	)
}

// NftToAdd represents a single NFT. There are validity constraints:
//    chain_code:        Min chars: 1, Max chars: 10, Characters allowed: ASCII a-z0-9, Case-insensitive
//    contract_address:  Min chars: 1, Max chars: 128
//    token_id:          Token ID of NFT. May be left blank if not applicable. Max 64 characters.
//    url:               URL of NFT asset, e.g. media url. May be left blank if not applicable. Max 128 characters.
//    hash:              SHA-256 hash of NFT asset, e.g. media url. May be left blank if not applicable. Max 64 characters.
//    metadata:          Serialized json, max. 64 chars. May be left empty.
type NftToAdd struct {
	ChainCode       string      `json:"chain_code"`
	ContractAddress string      `json:"contract_address"`
	TokenId         string      `json:"token_id"`
	Url             string      `json:"url"`
	Hash            string      `json:"hash"`
	Metadata        interface{} `json:"metadata"` // because this may change, it is an interface
}

// encodeMeta converts the Metadata field to an escaped json string
func (nft *NftToAdd) encodeMeta() nftEncoded {
	var md string
	if nft.Metadata != nil {
		j, e := json.Marshal(nft.Metadata)
		if e == nil && string(j) != `""` {
			md = fmt.Sprintf("%s", string(j))
		}
	}
	return nftEncoded{
		ChainCode:       nft.ChainCode,
		ContractAddress: nft.ContractAddress,
		TokenId:         nft.TokenId,
		Url:             nft.Url,
		Hash:            nft.Hash,
		Metadata:        md,
	}
}

// nftEncoded is what is serialized for the packed transaction, using an interface for NftToAdd.Metadata allows some flexibility
type nftEncoded struct {
	ChainCode       string `json:"chain_code"`
	ContractAddress string `json:"contract_address"`
	TokenId         string `json:"token_id"`
	Url             string `json:"url"`
	Hash            string `json:"hash"`
	Metadata        string `json:"metadata"`
}

// valid checks for various constraints.
func (nfte nftEncoded) valid() error {
	var chainCodeRex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	switch true {
	case !chainCodeRex.MatchString(nfte.ChainCode):
		return fmt.Errorf("chain code (%q) does not meet requirements: Min chars: 1, Max chars: 10, Characters allowed: ASCII a-z0-9, Case-insensitive", nfte.ChainCode)
	case len(nfte.ContractAddress) < 1 || len(nfte.ContractAddress) > 128:
		return fmt.Errorf("contract address (%q) does not meet requirements: Min chars: 1, Max chars: 128", nfte.ContractAddress)
	case len(nfte.TokenId) > 64:
		return fmt.Errorf("token id (%q) does not meet requirements: Max chars: 64", nfte.TokenId)
	case len(nfte.Url) > 128:
		return fmt.Errorf("url (%q) does not meet requirements: Max chars: 128", nfte.Url)
	}
	return nil
}

// AddNft is used to add an array of NFTs (max 3 per tx).
type AddNft struct {
	FioAddress string          `json:"fio_address"`
	Nfts       []NftToAdd      `json:"nfts"`
	MaxFee     uint64          `json:"max_fee"`
	Actor      eos.AccountName `json:"actor"`
	Tpid       string          `json:"tpid"`
}

// addNft is used to facilitate using an interface for metadata in NftToAdd
type addNft struct {
	FioAddress string          `json:"fio_address"`
	Nfts       []nftEncoded    `json:"nfts"`
	MaxFee     uint64          `json:"max_fee"`
	Actor      eos.AccountName `json:"actor"`
	Tpid       string          `json:"tpid"`
}

// NewAddNft creates an AddNft fio.Action
func NewAddNft(fioAddress string, nfts []NftToAdd, actor eos.AccountName) (*Action, error) {
	n := make([]nftEncoded, len(nfts))
	for i := range nfts {
		n[i] = nfts[i].encodeMeta()
	}
	add := &addNft{
		FioAddress: fioAddress,
		Nfts:       n,
		MaxFee:     Tokens(GetMaxFee(FeeAddNft)),
		Tpid:       CurrentTpid(),
		Actor:      actor,
	}
	if e := add.valid(); e != nil {
		return nil, e
	}
	return NewAction("fio.address", "addnft", actor, add), nil
}

// MustNewAddNft panics on error
func MustNewAddNft(fioAddress string, nfts []NftToAdd, actor eos.AccountName) *Action {
	a, e := NewAddNft(fioAddress, nfts, actor)
	if e != nil {
		panic(e)
	}
	return a
}

// valid checks for various constraints.
func (anft *addNft) valid() error {
	if !Address(anft.FioAddress).Valid() {
		return fmt.Errorf("fio address (%q) is invalid", anft.FioAddress)
	}
	if anft.Nfts == nil || len(anft.Nfts) > 3 || len(anft.Nfts) == 0 {
		return fmt.Errorf("min 1, max 3 nfts are required")
	}
	for _, n := range anft.Nfts {
		if e := n.valid(); e != nil {
			return e
		}
	}
	return nil
}

// NftToDelete is an individual NFT that should be removed from existing mappings, used by RemNft, The server will validate:
//    chain_code:       Min chars: 1, Max chars: 10, Characters allowed: ASCII a-z0-9, Case-insensitive
//    contract_address: Min chars: 1, Max chars: 128
//    token_id:         Token ID of NFT. May be left blank if not applicable. Max 64 characters.
type NftToDelete struct {
	ChainCode       string `json:"chain_code"`
	ContractAddress string `json:"contract_address"`
	TokenId         string `json:"token_id"`
}

// RemNft is used to remove NFTs from an address
type RemNft struct {
	FioAddress string          `json:"fio_address"`
	Nfts       []NftToDelete   `json:"nfts"`
	MaxFee     uint64          `json:"max_fee"`
	Actor      eos.AccountName `json:"actor"`
	Tpid       string          `json:"tpid"`
}

// NewRemNft creates an action for removing NFT mappings
func NewRemNft(fioAddress string, nfts []NftToDelete, actor eos.AccountName) (*Action, error) {
	if nfts == nil || len(nfts) == 0 {
		return nil, fmt.Errorf("nfts cannot be empty")
	}
	if !Address(fioAddress).Valid() {
		return nil, fmt.Errorf("invalid fio address")
	}
	for i := range nfts {
		switch true {
		case nfts[i].ChainCode == "" || len(nfts[i].ChainCode) > 10:
			return nil, fmt.Errorf("chain code (%q) must be > 1 and < 128 characters", nfts[i].ChainCode)
		case nfts[i].ContractAddress == "" || len(nfts[i].ContractAddress) > 128:
			return nil, fmt.Errorf("contract address (%q) must be > 1 and < 128 characters", nfts[i].ContractAddress)
		case len(nfts[i].TokenId) > 64:
			return nil, fmt.Errorf("token code must be (%q) < 64 characters", nfts[i].TokenId)
		}
	}
	return NewAction("fio.address", "remnft", actor, &RemNft{
		FioAddress: fioAddress,
		Nfts:       nfts,
		MaxFee:     Tokens(GetMaxFee(FeeRemoveNft)),
		Actor:      actor,
		Tpid:       CurrentTpid(),
	}), nil
}

// MustNewRemNft creates an action or panics
func MustNewRemNft(fioAddress string, nfts []NftToDelete, actor eos.AccountName) *Action {
	a, e := NewRemNft(fioAddress, nfts, actor)
	if e != nil {
		panic(e)
	}
	return a
}

// RemAllNft removes all NFTs for a FIO Address
type RemAllNft struct {
	FioAddress string          `json:"fio_address"`
	MaxFee     uint64          `json:"max_fee"`
	Actor      eos.AccountName `json:"actor"`
	Tpid       string          `json:"tpid"`
}

// NewRemAllNft builds an action for RemAllNft
func NewRemAllNft(fioAddress string, actor eos.AccountName) *Action {
	return NewAction("fio.address", "remallnfts", actor, &RemAllNft{
		FioAddress: fioAddress,
		MaxFee:     Tokens(GetMaxFee(FeeRemoveAllNfts)),
		Actor:      actor,
		Tpid:       CurrentTpid(),
	})
}

type Nft struct {
	ChainCode       string `json:"chain_code,omitempty"`
	ContractAddress string `json:"contract_address,omitempty"`
	TokenId         string `json:"token_id,omitempty"`
	Url             string `json:"url,omitempty"`
	Hash            string `json:"hash,omitempty"`
	Metadata        string `json:"metadata,omitempty"`
}

type NftResponse struct {
	Nfts []Nft  `json:"nfts"`
	More uint32 `json:"more"`
}

type getNftsReq struct {
	FioAddress      string `json:"fio_address,omitempty"`
	ChainCode       string `json:"chain_code,omitempty"`
	ContractAddress string `json:"contract_address,omitempty"`
	TokenId         string `json:"token_id,omitempty"`
	Hash            string `json:"hash,omitempty"`
	Limit           uint32 `json:"limit,omitempty"`
	Offset          uint32 `json:"offset,omitempty"`
}

// GetNftsFioAddress fetches the list of NFTs for a FIO address
func (api *API) GetNftsFioAddress(fioAddress string, offset uint32, limit uint32) (nfts *NftResponse, err error) {
	nfts = &NftResponse{
		Nfts: make([]Nft, 0),
	}
	err = api.call("chain", "get_nfts_fio_address", getNftsReq{FioAddress: fioAddress, Limit: limit, Offset: offset}, nfts)
	return
}

// GetNftsContract fetches the list of NFTs for a contract address
func (api *API) GetNftsContract(chaincode, contractAddress, tokenid string, offset uint32, limit uint32) (nfts *NftResponse, err error) {
	nfts = &NftResponse{
		Nfts: make([]Nft, 0),
	}
	err = api.call("chain", "get_nfts_contract", getNftsReq{
		ChainCode:       chaincode,
		ContractAddress: contractAddress,
		TokenId:         tokenid,
		Limit:           limit,
		Offset:          offset,
	}, nfts)
	return
}

// GetNftsHash fetches the list of NFTs for a specific NFT Hash
func (api *API) GetNftsHash(hash string, offset uint32, limit uint32) (nfts *NftResponse, err error) {
	nfts = &NftResponse{
		Nfts: make([]Nft, 0),
	}
	err = api.call("chain", "get_nfts_hash", getNftsReq{Hash: hash, Limit: limit, Offset: offset}, nfts)
	return
}
