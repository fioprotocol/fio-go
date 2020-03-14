package fio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mitchellh/go-ps"
	"io/ioutil"
	"net"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

type KeosClient struct {
	BaseUrl    string
	HttpClient *http.Client
	Socket     string
	Keys       map[string]KeosKeys `json:"-"`
	Wallet     string
	password   string
}

type KeosKeys struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
	FioAddress string `json:"fio_address"`
}

func NewKeosClient(keosUrl string, socket string) *KeosClient {
	client := &KeosClient{}
	client.BaseUrl = "http://unix"
	client.HttpClient = &http.Client{}
	client.Keys = make(map[string]KeosKeys)
	// by default we use a unix socket in the user's home directory:
	if keosUrl == "" {
		client.HttpClient = &http.Client{
			Transport: &http.Transport{
				IdleConnTimeout:    3 * time.Second,
				DisableCompression: true,
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", socket)
				},
			},
		}
	} else {
		client.BaseUrl = keosUrl
		client.HttpClient = &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:       1,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: true,
			},
		}
	}
	return client
}

type alreadyUnlocked struct {
	Error struct {
		What string `json:"what"`
	} `json:"error"`
}

func (k *KeosClient) Unlock(password string, wallet string) error {
	k.Wallet = wallet
	k.password = password
	if password != "" {
		resp, err := k.HttpClient.Post(k.BaseUrl+"/v1/wallet/unlock", "application/json", bytes.NewReader([]byte(`["`+k.Wallet+`","`+k.password+`"]`)))
		if err != nil {
			return err
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			already := &alreadyUnlocked{}
			e := json.Unmarshal(body, already)
			if e == nil && already.Error.What == "Already unlocked" {
				// not a problem, already unlocked
				return nil
			}
			j, e := json.MarshalIndent(json.RawMessage(body), "", "  ")
			if e != nil {
				return errors.New("could not open wallet. Ensure it is unlocked, or use '-p' to provide the password")
			}
			return errors.New(string(j))
		}
	} else {
		return errors.New("password not supplied, '-password' option is mandatory")
	}
	return nil
}

func (k KeosClient) Start(noKeosd bool) error {
	if noKeosd {
		return nil
	}
	//cmd := exec.Command("keosd", "--http-server-address", "--https-server-address", "--unix-socket-path", "keosd.sock")
	cmd := exec.Command("clio", "wallet", "list") // let clio start keosd
	_ = cmd.Run()                                 // ignore output
	var isRunning bool
	procs, _ := ps.Processes()
	for _, p := range procs {
		if p.Executable() == "keosd" {
			isRunning = true
			break
		}
	}
	if !isRunning {
		return errors.New("could not verify keosd is running")
	}
	return nil
}

func firstName(pubkey string, api *API) string {
	names, found, err := api.GetFioNames(pubkey)
	if err != nil || !found {
		return ""
	}
	return names.FioAddresses[0].FioAddress
}

func (k *KeosClient) GetKeys(nodeosApi *API) error {
	// get a list of available keys:
	resp, err := k.HttpClient.Post(k.BaseUrl+"/v1/wallet/list_keys", "application/json", bytes.NewReader([]byte(`["`+k.Wallet+`","`+k.password+`"]`)))
	if err != nil {
		return errors.New("could not connect to keosd, is the wallet unlocked?\n" + err.Error())
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		j, e := json.MarshalIndent(json.RawMessage(body), "", "  ")
		if e != nil {
			return errors.New("could not open wallet. Ensure it is unlocked, or use '-p' to provide the password")
		}
		return errors.New(string(j))
	}

	// build a map of available keys by actor:
	pubKeys := make([][]string, 0)
	_ = json.Unmarshal(body, &pubKeys)
	if len(pubKeys) == 0 {
		return errors.New("no keys found in the wallet")
	}
	mux := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(pubKeys))
	for _, pk := range pubKeys {
		go func(pk []string) {
			defer wg.Done()
			a, e := ActorFromPub(pk[0])
			if e != nil {
				return
			}
			first := firstName(pk[0], nodeosApi)
			mux.Lock()
			k.Keys[string(a)] = KeosKeys{
				PublicKey:  pk[0],
				PrivateKey: pk[1],
				FioAddress: first,
			}
			mux.Unlock()
		}(pk)
	}
	wg.Wait()
	return nil
}

func (k *KeosClient) PrintKeys() string {
	buf := bytes.NewBufferString("")
	buf.WriteString(fmt.Sprintf("\n%-12s  %-53s  %s\n", "account", "Public Key", "FIO Address"))
	buf.WriteString(fmt.Sprintf("%-12s  %-53s  %s\n", "-------", "----------", "-----------"))
	for k, v := range k.Keys {
		buf.WriteString(fmt.Sprintf("%12s  %53s  %s\n", k, v.PublicKey, v.FioAddress))
	}
	return buf.String()
}
