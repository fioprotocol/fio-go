package fio

import (
	"fmt"
	"math/rand"
	"net/http"
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
	_, err = api.SignPushActions(
		NewTransferTokensPubKey(
			account.Actor,
			voter.PubKey,
			Tokens(GetMaxFee(FeeRegisterProxy)+10.0),
		).ToEos(),
	)
	if err != nil {
		t.Error(err)
		return
	}
	rand.Seed(time.Now().UnixNano())
	fioAddr := "vote-" + word() + "@dapixdev"
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

	_, err = voterApi.SignPushActions(NewRegProxy(fioAddr, voter.Actor).ToEos())
	if err != nil {
		t.Error(err)
		return
	}

	pVoter, _ := NewRandomAccount()
	_, err = api.SignPushActions(
		NewTransferTokensPubKey(
			account.Actor,
			pVoter.PubKey,
			Tokens(1.0),
		).ToEos(),
	)
	if err != nil {
		t.Error(err)
		return
	}
	rand.Seed(time.Now().UnixNano())
	pFioAddr := "proxyvote-" + word() + "@dapixdev"
	_, err = api.SignPushActions(MustNewRegAddress(
		account.Actor, Address(pFioAddr), pVoter.PubKey).ToEos(),
	)
	if err != nil {
		t.Error(err)
		return
	}

	pApi, _, err := NewConnection(pVoter.KeyBag, api.BaseURL)
	if err != nil {
		t.Error(err)
		return
	}
	_, err = pApi.SignPushActions(NewVoteProxy(fioAddr, pFioAddr, pVoter.Actor).ToEos())
	if err != nil {
		t.Error(err)
		return
	}

}

func TestAPI_GetProducerSchedule(t *testing.T) {
	_, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}
	sched, err := api.GetProducerSchedule()
	if err != nil {
		t.Error(err)
		return
	}
	if len(sched.Active.Producers) == 0 {
		t.Error("got empty producer schedule")
	}
}

func TestAPI_Register_GetBpJson(t *testing.T) {
	account, api, _, err := newApi()
	if err != nil {
		t.Error(err)
		return
	}

	prod, _ := NewRandomAccount()
	_, err = api.SignPushActions(
		NewTransferTokensPubKey(
			account.Actor,
			prod.PubKey,
			Tokens(GetMaxFee(FeeRegisterProducer)+GetMaxFee(FeeUnregisterProducer)+10.0),
		).ToEos(),
	)
	if err != nil {
		t.Error(err)
		return
	}
	rand.Seed(time.Now().UnixNano())
	fioAddr := "producer-" + word() + "@dapixdev"
	_, err = api.SignPushActions(MustNewRegAddress(
		account.Actor, Address(fioAddr), prod.PubKey).ToEos(),
	)
	if err != nil {
		t.Error(err)
		return
	}

	// start a mock server
	u := make(chan string, 1)
	var url string
	go servBpJson(u)
	select {
	case <-time.After(2 * time.Second):
		t.Error("mock server did not start")
		return
	case url = <-u:
	}
	fmt.Println(url)

	prodApi, _, err := NewConnection(prod.KeyBag, api.BaseURL)
	if err != nil {
		t.Error(err)
		return
	}

	nrp, err := NewRegProducer(fioAddr, prod.PubKey, "http://"+url, LocationEastAsia, prod.Actor)
	if err != nil {
		t.Error(err)
		return
	}
	_, err = prodApi.SignPushActions(nrp.ToEos())
	if err != nil {
		t.Error(err)
		return
	}

	// try to unreg on exit, no matter what, so a dev node doesn't end up with an insufficient
	// number of working producers to maintain quorum
	defer func() {
		_, _ = prodApi.SignPushActions(NewUnRegProducer(fioAddr, prod.Actor).ToEos())
	}()

	_, err = api.GetBpJson(prod.Actor)
	if err == nil {
		t.Error("private IP in producer table should fail")
	}

	// bypasses anti-ssrf checks:
	gbj, err := api.getBpJson(prod.Actor, true)
	if err != nil {
		t.Error(err)
		return
	}
	if gbj.ProducerAccountName != "test" {
		t.Error("could not get bp.json info")
	}
}

// mock http server
func servBpJson(listen chan string) {
	rand.Seed(time.Now().UnixNano())
	port := rand.Intn(32768) + 32767

	bpJson := []byte(`{
	  "producer_account_name": "test",
	  "org": {
	    "candidate_name": "",
	    "website": "",
	    "code_of_conduct":"",
	    "ownership_disclosure":"",
	    "email":"",
	    "branding":{
	      "logo_256":"",
	      "logo_1024":"",
	      "logo_svg":""
	    },
	    "location": {
	      "name": "",
	      "country": "",
	      "latitude": 0,
	      "longitude": 0
	    },
	    "social": {
	      "steemit": "",
	      "twitter": "",
	      "youtube": "",
	      "facebook": "",
	      "github":"",
	      "reddit": "",
	      "keybase": "",
	      "telegram": "",
	      "wechat":""
	    }
	  },
	  "nodes": [
	    {
	      "location": {
	        "name": "",
	        "country": "",
	        "latitude": 0,
	        "longitude": 0
	      },
	      "node_type": "producer",
	      "p2p_endpoint": "",
	      "bnet_endpoint": "",
	      "api_endpoint": "",
	      "ssl_endpoint": ""
	    },
	    {
	      "location": {
	        "name": "",
	        "country": "",
	        "latitude": 0,
	        "longitude": 0
	      },
	      "node_type":"seed",
	      "p2p_endpoint": "",
	      "bnet_endpoint": "",
	      "api_endpoint": "",
	      "ssl_endpoint": ""
	    }
	  ]
	}`)

	respond := func(resp http.ResponseWriter, req *http.Request) {
		req.Body.Close()
		resp.WriteHeader(http.StatusOK)
		resp.Write(bpJson)
	}

	http.HandleFunc(`/`, respond)
	url := fmt.Sprintf("127.0.0.1:%d", port)
	fmt.Println("start mock bp.json server on " + url)

	listen <- url
	panic(http.ListenAndServe(url, nil))
}
