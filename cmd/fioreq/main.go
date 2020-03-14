package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/dapixio/fio-go"
	"github.com/eoscanada/eos-go"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {

	var (
		noKeosd    bool
		socket     string
		keosUrl    string
		nodeosUrl  string
		password   string
		wallet     string
		permission string
		command    string
		requestId  string
		limit      int
		offset     int
		fullResp   bool
	)

	homeDir, _ := os.UserHomeDir()
	if homeDir == "" {
		homeDir = os.Getenv("HOME")
		if homeDir == "" {
			homeDir = "/"
		}
	}
	socket = fmt.Sprintf("%s%ceosio-wallet%ckeosd.sock", homeDir, os.PathSeparator, os.PathSeparator)

	flag.IntVar(&limit, "limit", 10, "max count of records to fetch (optional: pending, sent")
	flag.IntVar(&offset, "offset", 0, "starting record for fetch (optional: pending, sent")
	flag.StringVar(&requestId, "id", "", "FIO request id (required: view-req, reject; optional: record)")
	flag.StringVar(&nodeosUrl, "u", "http://127.0.0.1:8888", "http url for nodeos")
	flag.StringVar(&socket, "wallet", socket, "unix domain socket for keosd api")
	flag.StringVar(&keosUrl, "keosd", "", "TCP url for keosd api (http://...), overrides 'wallet' flag")
	flag.StringVar(&password, "password", "", "(required) password to unlock wallet")
	flag.StringVar(&wallet, "n", "default", "wallet name")
	flag.StringVar(&permission, "p", "", "(required for -c: show, reject, request, record) actor used for operations")
	flag.StringVar(&command, "c", "", "command to run")
	flag.BoolVar(&noKeosd, "no-auto-keosd", false, "Don't try to launch a keosd")
	flag.BoolVar(&fullResp, "full", false, "print full transaction result traces")
	flag.Parse()

	printErr := func() {
		fmt.Println("fioreq is a utility for interacting with funds requests on the FIO blockchain.")
		fmt.Println("Usage:\n\tfioreq -password <password> -c <command> <options>")
		fmt.Println("\nuse 'fioreq -h' to show other options.")
		fmt.Println("Commands:")
		fmt.Printf("  -c %9s - list available accounts\n", "list")
		fmt.Printf("  -c %9s - show pending FIO requests\n", "pending")
		fmt.Printf("  -c %9s - show sent FIO requests\n", "sent")
		fmt.Printf("  -c %9s - view a FIO request\n", "view-req")
		fmt.Printf("  -c %9s - view a FIO response\n", "view-resp")
		fmt.Printf("  -c %9s - reject request\n", "reject")
		fmt.Printf("  -c %9s - send a request\n", "request")
		fmt.Printf("  -c %9s - record transaction\n", "record")
		os.Exit(2)
	}

	if command == "" {
		printErr()
	}

	if password == "" && command != "pending" && command != "sent" {
		fmt.Print("Please enter keosd password: ")
		b, err := terminal.ReadPassword(0)
		if err != nil {
			fmt.Println("")
			log.Println(err)
		}
		password = string(b)
		fmt.Println("")
		if password == "" {
			log.Fatal("password is required.")
		}
	}

	must := func(e error) {
		if e != nil {
			log.Fatal(e)
		}
	}

	keosd := fio.NewKeosClient(keosUrl, socket)
	must(keosd.Start(noKeosd))
	if command != "pending" && command != "sent" {
		must(keosd.Unlock(password, wallet))
	}

	// ensure we can connect to nodeos, use the read only api for lookups:
	roApi, _, err := fio.NewConnection(nil, nodeosUrl)
	if err != nil {
		log.Println("Could not connect to nodeos server")
		log.Fatal(err)
	}

	// get a list of available keys:
	if command != "pending" && command != "sent" {
		must(keosd.GetKeys(roApi))
	}

	actorRequired := func() {
		if permission == "" {
			log.Fatalf("the command '%s' requires an actor permission, please supply an actor using '-p'", command)
		}
	}

	requestIdRequired := func() uint64 {
		if requestId == "" {
			log.Fatalf("command %s requires a FIO request ID specified with the '-id' flag", command)
		}
		i, err := strconv.ParseInt(requestId, 10, 64)
		if err != nil {
			log.Fatalf("command %s requires a FIO request ID specified with the '-id' flag: %s", command, err.Error())
		}
		return uint64(i)
	}

	get := func(){
		actorRequired()
		pending, err := printSentPending(command, permission, roApi, limit, offset)
		must(err)
		fmt.Println(pending)
		os.Exit(0)
	}

	switch command {
	case "pending":
		get()
	case "sent":
		get()
	case "list":
		fmt.Println(keosd.PrintKeys())
		os.Exit(0)
	case "view-req":
		outer, resp, _, err := view(requestIdRequired(), permission, keosd, nodeosUrl)
		if outer != nil {
			fmt.Println("\nRequest:")
			fmt.Println(strings.Repeat("⎺", 86))
			ppYaml(outer)
		}
		must(err)
		fmt.Println("\nDecrypted Content:")
		fmt.Println(strings.Repeat("⎺", 86))
		ppYaml(resp)
		os.Exit(0)
	case "reject":
		ok, resp, err := reject(requestIdRequired(), permission, keosd, nodeosUrl)
		must(err)
		if fullResp {
			rawJsonPrintLn(resp)
			os.Exit(0)
		}
		fmt.Println(ok)
		os.Exit(0)
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printErr()
	}
}

func ppYaml(y []byte){
	lines := strings.Split(string(y), "\n")
	for _, l := range lines {
		cols := strings.Split(l, ": ")
		if len(cols) == 2 {
			if len(cols[1]) > 64 {
				cols[1] = cols[1][:61]+"..."
			}
			cols[1] = strings.TrimPrefix(cols[1], `"`)
			cols[1] = strings.TrimSuffix(cols[1], `"`)
			fmt.Printf("%19s   %s\n", cols[0], cols[1])
		} else {
			fmt.Println(l)
		}
	}
}

func rawJsonPrintLn(raw json.RawMessage){
	if len(raw) < 2 {
		fmt.Println("Empty result.")
		return
	}
	j, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(j))
}

func printSentPending(which string, actor string, api *fio.API, limit int, offset int) (results string, err error) {
	if len(actor) != 12 {
		return "", errors.New("invalid actor, expected 12 character string")
	}
	pubkey, err := getPubForActor(actor, api)
	if err != nil {
		return "", err
	}

	var (
		requests fio.PendingFioRequestsResponse
		has bool
		title string
	)
	switch which {
	case "pending":
		title = "Pending FIO Requests:\n\n"
		requests, has, err = api.GetPendingFioRequests(pubkey, limit, offset)

	case "sent":
		title = "Sent FIO Requests:\n\n"
		requests, has, err = api.GetSentFioRequests(pubkey, limit, offset)
	}
	if err != nil {
		return "", err
	}
	if !has {
		return "No pending requests", nil
	}

	buf := bytes.NewBufferString(title)
	buf.WriteString(fmt.Sprintf("%-19s %3s %-6s %16s  %-16s\n", "Date", "", "ID", "From", "To"))
	buf.WriteString(strings.Repeat("⎺", 64)+"\n")
	for _, req := range requests.Requests {
		f := req.PayeeFioAddress
		t := req.PayerFioAddress
		if len(f) > 16 {
			f = f[:13]+"..."
		}
		if len(t) > 16 {
			t = t[:13]+"..."
		}
		status := ""
		//hasResp, response, _ := api.GetFioRequestStatus(req.FioRequestId)
			switch req.Status {
			case "rejected":
				status = "✘ "
			case "sent_to_blockchain":
				status = "︎ ✔ "
			}
		buf.WriteString(fmt.Sprintf("%19s %3s %-6d %16s  %-16s\n",req.TimeStamp.Time.Local().Format(time.RFC822), status, req.FioRequestId, f, t))
	}
	if requests.More > 0 {
		buf.WriteString(fmt.Sprintf("\n%d additional pending requests.", requests.More))
	}
	return buf.String(), nil
}

type aMap struct {
	Clientkey string `json:"clientkey"`
}

//FIXME: GetFioAccount is returning a bad result, for now lookup pubkey in table:
func getPubForActor(actor string, api *fio.API) (pubkey string, err error) {
	name, err := eos.StringToName(actor)
	if err != nil {
		return "", err
	}
	resp, err := api.GetTableRows(eos.GetTableRowsRequest{
		Code:       "fio.address",
		Scope:      "fio.address",
		Table:      "accountmap",
		LowerBound: fmt.Sprintf("%d", name),
		UpperBound: fmt.Sprintf("%d", name),
		Limit:      1,
		KeyType:    "i64",
		Index:      "1",
		JSON:       true,
	})
	if err != nil {
		fmt.Println(err)
	}
	found := make([]aMap, 0)
	err = json.Unmarshal(resp.Rows, &found)
	if err != nil {
		return "", err
	}
	if len(found) == 0 || found[0].Clientkey == "" {
		return "", errors.New("no matching account found in fio.address accountmap table")
	}
	return found[0].Clientkey, nil
}

func authenticate(actor string, keosd *fio.KeosClient, nodeosUrl string) (account *fio.Account, api *fio.API, opts *fio.TxOptions, err error) {
	if keosd.Keys == nil || keosd.Keys[actor].PrivateKey == "" {
		err = errors.New("could not find private key for actor "+actor)
		return
	}
	account, err = fio.NewAccountFromWif(keosd.Keys[actor].PrivateKey)
	if err != nil {
		return
	}
	api, opts, err = fio.NewConnection(account.KeyBag, nodeosUrl)
	if err != nil {
		return
	}
	return
}

func reject(requestId uint64, actor string, keosd *fio.KeosClient, nodeosUrl string) (ok bool, resp json.RawMessage, err error) {
	_, api, opts, err := authenticate(actor, keosd, nodeosUrl)
	if err != nil {
		return
	}
	_, tx, err := api.SignTransaction(
		fio.NewTransaction([]*fio.Action{fio.NewRejectFndReq(eos.AccountName(actor), fmt.Sprintf("%d", requestId))}, opts),
		opts.ChainID, fio.CompressionNone,
	)
	if err != nil {
		return
	}
	resp, err = api.PushTransactionRaw(tx)
	if err == nil {
		ok = true
	}
	return
}

func view(requestId uint64, actor string, keosd *fio.KeosClient, nodeosUrl string) (outer []byte, resp []byte, record []byte,  err error) {
	account, api, _, err := authenticate(actor, keosd, nodeosUrl)
	if err != nil {
		return
	}
	request, err := api.GetFioRequest(requestId)
	if err != nil {
		return
	}
	outer, err = yaml.Marshal(request)
	if err != nil {
		return
	}
	if request.PayerKey != account.PubKey && request.PayeeKey != account.PubKey {
		return outer, nil, nil, errors.New("actor cannot decrypt this request, it is not a party to the transaction")
	}
	counterParty := request.PayeeKey
	if account.PubKey == request.PayeeKey {
		counterParty = request.PayerKey
	}

	result, err := fio.DecryptContent(account, counterParty, request.Content, fio.ObtRequestType)
	if err != nil {
		return
	}
	resp, err = yaml.Marshal(result.Request)
	return
}

