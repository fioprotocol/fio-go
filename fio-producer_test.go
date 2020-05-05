package fio

import (
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestNewVoteProducer(t *testing.T) {
	account, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}

	voter, _ := NewRandomAccount()
	_, err = api.SignPushActions(NewTransferTokensPubKey(
		account.Actor,
		voter.PubKey,
		Tokens(GetMaxFee(FeeVoteProducer))+1.0).ToEos(),
	)
	if err != nil {
		t.Error(err)
		return
	}
	rand.Seed(time.Now().UnixNano())
	fioAddr := "vote-"+word()+"@dapixdev"
	_, err = api.SignPushActions(MustNewRegAddress(
		account.Actor, Address(fioAddr), voter.PubKey).ToEos(),
	)
	if err != nil {
		t.Error(err)
		return
	}

	prods, err := api.GetFioProducers()
	if err != nil {
		t.Error(err)
		return
	}
	if len(prods.Producers) == 0 {
		t.Error("didn't get producer list")
	}
	newVotes := make([]string, 0)
	for _, v := range prods.Producers {
		if v.IsActive == 1 {
			newVotes = append(newVotes, string(v.FioAddress))
		}
		if len(newVotes) >= 30 {
			break
		}
	}
	sort.Strings(newVotes)

	voterApi, _, err := NewConnection(voter.KeyBag, api.BaseURL)
	if err != nil {
		t.Error(err)
		return
	}

	_, err = voterApi.SignPushActions(
		NewVoteProducer(newVotes, voter.Actor, fioAddr).ToEos(),
	)
	if err != nil {
		t.Error(err)
		return
	}

	existingVotes, err := api.GetVotes(string(voter.Actor))
	if err != nil {
		t.Error(err)
		return
	}
	sort.Strings(existingVotes)

	if strings.Join(newVotes, "") != strings.Join(existingVotes, "") {
		t.Error("votes not updated")
	}

}
