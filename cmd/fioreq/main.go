package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/eoscanada/eos-go"
	"github.com/fioprotocol/fio-go"
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
		payer      string
		payee      string
		limit      int
		offset     int
		fullResp   bool
		template   bool
		example    bool
	)

	homeDir, _ := os.UserHomeDir()
	if homeDir == "" {
		homeDir = os.Getenv("HOME")
		if homeDir == "" {
			homeDir = "/"
		}
	}
	socket = fmt.Sprintf("%s%ceosio-wallet%ckeosd.sock", homeDir, os.PathSeparator, os.PathSeparator)

	flag.IntVar(&limit, "limit", 50, "max count of records to fetch (optional: pending, sent")
	flag.IntVar(&offset, "offset", 0, "starting record for fetch (optional: pending, sent")
	flag.StringVar(&requestId, "id", "", "FIO request id (required: view-req, reject; optional: record)")
	flag.StringVar(&nodeosUrl, "u", "http://127.0.0.1:8888", "http url for nodeos")
	flag.StringVar(&socket, "wallet", socket, "unix domain socket for keosd api")
	flag.StringVar(&keosUrl, "keosd", "", "TCP url for keosd api (http://...), overrides 'wallet' flag")
	flag.StringVar(&password, "password", "", "(required) password to unlock wallet")
	flag.StringVar(&wallet, "n", "default", "wallet name")
	flag.StringVar(&permission, "p", "", "(required for -c: show, reject, request, record) actor used for operations")
	flag.StringVar(&command, "c", "", "command to run")
	flag.StringVar(&payer, "payer", "", "payer for new funds request")
	flag.StringVar(&payee, "payee", "", "payee (your address) for new funds request")
	flag.BoolVar(&noKeosd, "no-auto-keosd", false, "Don't try to launch a keosd")
	flag.BoolVar(&fullResp, "full", false, "print full transaction result traces")
	flag.BoolVar(&template, "template", false, "print JSON templates for funds request, and response")
	flag.BoolVar(&example, "example", false, "show example commands")
	flag.Parse()

	if example {
		printExample()
		os.Exit(0)
	}
	if template {
		printTemplate()
		os.Exit(0)
	}

	printErr := func() {
		fmt.Println("fioreq is a utility for interacting with funds requests on the FIO blockchain.")
		fmt.Println("Usage:\n\tfioreq -password <password> -c <command> <options> <request/response request>")
		fmt.Println("\nuse 'fioreq -h' to show other options.")
		fmt.Println("Commands:")
		fmt.Printf("  -c %9s - list available accounts\n", "list")
		fmt.Printf("  -c %9s - show pending FIO requests\n", "pending")
		fmt.Printf("  -c %9s - show sent FIO requests\n", "sent")
		fmt.Printf("  -c %9s - view a FIO request\n", "view-req")
		//TODO: add ability for getting response without a corresponding request.
		//fmt.Printf("  -c %9s - view a FIO response\n", "view-resp")
		fmt.Printf("  -c %9s - reject request\n", "reject")
		fmt.Printf("  -c %9s - send a new request\n", "request")
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

	jsonRequired := func() string {
		ok := true
		if flag.NArg() != 1 {
			ok = false
		}
		js := strings.TrimSpace(flag.Arg(0))
		if !strings.HasPrefix(js, "{") || !strings.HasSuffix(js, "}") {
			ok = false
		}
		jrm := json.RawMessage([]byte(js))
		j, err := json.Marshal(&jrm)
		if err != nil {
			fmt.Println(err)
			ok = false
		}
		if !ok {
			fmt.Print("\nError, a JSON document must be supplied, see the following examples\n\n")
			printExample()
			os.Exit(2)
		}
		return string(j)
	}

	get := func() {
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
		outer, resp, pubk, err := view(requestIdRequired(), permission, keosd, nodeosUrl)
		if outer != nil {
			ppYaml("Request", outer)
		}
		must(err)
		ppYaml("Decrypted Content", resp)
		hasResp, recordObt, _ := viewRecord(requestIdRequired(), permission, pubk, keosd, nodeosUrl)
		if hasResp && string(recordObt) != "" {
			ppYaml("Response", recordObt)
			fmt.Println("")
		}
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
	case "request":
		actorRequired()
		id, resp, err := requestNew(payer, payee, permission, jsonRequired(), keosd, nodeosUrl)
		if err != nil {
			fmt.Println("New Funds request failed:\n" + err.Error())
			os.Exit(1)
		}
		if id == "" {
			fmt.Println("transaction failed.")
			os.Exit(1)
		}
		if fullResp {
			rawJsonPrintLn(resp)
			os.Exit(0)
		}
		fmt.Println("success, transaction id: " + id)
		os.Exit(0)
	case "record":
		actorRequired()
		id, resp, err := recordObt(requestIdRequired(), payer, payee, permission, jsonRequired(), keosd, nodeosUrl)
		if err != nil {
			fmt.Println("Record transaction failed:\n" + err.Error())
			os.Exit(1)
		}
		if id == "" {
			fmt.Println("transaction failed.")
			os.Exit(1)
		}
		if fullResp {
			rawJsonPrintLn(resp)
			os.Exit(0)
		}
		fmt.Println("success, transaction id: " + id)
		os.Exit(0)
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printErr()
	}
}

func recordObt(requestId uint64, payer string, payee string, actor string, requestJson string, keosd *fio.KeosClient, nodeosUrl string) (txid string, results json.RawMessage, err error) {
	if requestId == 0 {
		err = errors.New("must supply a request id")
	}
	account, api, opts, err := authenticate(actor, keosd, nodeosUrl)
	if err != nil {
		return
	}
	request, err := api.GetFioRequest(requestId)
	if err != nil {
		return
	}
	req := &fio.ObtRecordContent{}
	err = json.Unmarshal([]byte(requestJson), req)
	if err != nil {
		return
	}
	pubKey := request.PayeeKey
	if pubKey == account.PubKey {
		pubKey = request.PayerKey
	}
	content, err := req.Encrypt(account, pubKey)
	if err != nil {
		return
	}
	id := strconv.Itoa(int(requestId))
	_, tx, err := api.SignTransaction(
		fio.NewTransaction([]*fio.Action{
			fio.NewRecordSend(account.Actor, id, payer, payee, content),
		}, opts), opts.ChainID, fio.CompressionZlib,
	)
	if err != nil {
		return
	}
	results, err = api.PushTransactionRaw(tx)
	if err != nil {
		return
	}
	res := &eos.PushTransactionFullResp{}
	err = json.Unmarshal(results, res)
	if err != nil {
		show := 256
		if len(results) < show {
			show = len(results)
		}
		err = errors.New(fmt.Sprintf("could not decode transaction result, showing first %d bytes of response: %s\n%s\n", show, err.Error(), string(results[:show])))
		return
	}
	txid = res.TransactionID
	return
}

func requestNew(payer string, payee string, actor string, requestJson string, keosd *fio.KeosClient, nodeosUrl string) (txid string, results json.RawMessage, err error) {
	if !fio.Address(payer).Valid() {
		err = errors.New("payer address is invalid")
	}
	if !fio.Address(payee).Valid() {
		err = errors.New("payee address is invalid")
	}

	account, api, opts, err := authenticate(actor, keosd, nodeosUrl)
	if err != nil {
		return
	}
	pubKey, found, err := api.PubAddressLookup(fio.Address(payer), "FIO", "FIO")
	if err != nil {
		return
	}
	if !found {
		err = errors.New("could not get a public key for the FIO address: " + payer)
		return
	}
	req := &fio.ObtRequestContent{}
	err = json.Unmarshal([]byte(requestJson), req)
	if err != nil {
		return
	}

	content, err := req.Encrypt(account, pubKey.PublicAddress)
	if err != nil {
		return
	}
	_, tx, err := api.SignTransaction(
		fio.NewTransaction([]*fio.Action{
			fio.NewFundsReq(account.Actor, payer, payee, content),
		}, opts), opts.ChainID, fio.CompressionZlib,
	)
	if err != nil {
		return
	}
	results, err = api.PushTransactionRaw(tx)
	if err != nil {
		return
	}
	res := &eos.PushTransactionFullResp{}
	err = json.Unmarshal(results, res)
	if err != nil {
		show := 256
		if len(results) < show {
			show = len(results)
		}
		err = errors.New(fmt.Sprintf("could not decode transaction result, showing first %d bytes of response: %s\n%s\n", show, err.Error(), string(results[:show])))
		return
	}
	txid = res.TransactionID
	return
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
		has      bool
		title    string
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
	buf.WriteString(strings.Repeat("⎺", 64) + "\n")
	for _, req := range requests.Requests {
		f := req.PayeeFioAddress
		t := req.PayerFioAddress
		if len(f) > 16 {
			f = f[:13] + "..."
		}
		if len(t) > 16 {
			t = t[:13] + "..."
		}
		status := ""
		//hasResp, response, _ := api.GetFioRequestStatus(req.FioRequestId)
		switch req.Status {
		case "rejected":
			status = "✘ "
		case "sent_to_blockchain":
			status = "︎ ✔︎ "
		}
		buf.WriteString(fmt.Sprintf("%19s %3s %-6d %16s  %-16s\n", req.TimeStamp.Time.Local().Format(time.RFC822), status, req.FioRequestId, f, t))
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
		err = errors.New("could not find private key for actor " + actor)
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

func view(requestId uint64, actor string, keosd *fio.KeosClient, nodeosUrl string) (outer []byte, resp []byte, counterParty string, err error) {
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
		return outer, nil, "", errors.New("actor cannot decrypt this request, it is not a party to the transaction")
	}
	counterParty = request.PayeeKey
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

func viewRecord(requestId uint64, actor string, counterParty string, keosd *fio.KeosClient, nodeosUrl string) (found bool, record []byte, err error) {
	account, api, _, err := authenticate(actor, keosd, nodeosUrl)
	if err != nil {
		return
	}
	found, request, err := api.GetFioRequestStatus(requestId)
	switch {
	case err != nil || !found:
		return
	case request.Status == 1:
		record = []byte("Rejected: Request was rejected at " + time.Unix(int64(request.TimeStamp/1000000), 0).Local().Format(time.UnixDate))
	case counterParty == "":
		record = []byte("Error: cannot decrypt request, no public key for payer.")
	case request.Metadata != "":
		decrypted := &fio.ObtContentResult{}
		decrypted, err = fio.DecryptContent(account, counterParty, request.Metadata, fio.ObtResponseType)
		if err != nil {
			return
		}
		record, _ = yaml.Marshal(decrypted.Record)
		if request.TimeStamp != 0 {
			record = append(record, []byte(fmt.Sprintf(`time: "%s"`, time.Unix(int64(request.TimeStamp/1000000), 0).Local().Format(time.UnixDate)))...)
		}
	}
	return
}

func fixupFieldName(s string) string {
	switch s {
	case "amount":
		return "Amount"
	case "chaincode":
		return "Chain Code"
	case "content":
		return "Content"
	case "fiorequestid":
		return "Fio Request ID"
	case "hash":
		return "Hash"
	case "memo":
		return "Memo"
	case "obtid":
		return "OBT ID"
	case "offlineurl":
		return "Offline URL"
	case "payeefioaddress":
		return "Payee FIO Address"
	case "payeekey":
		return "Payee Key"
	case "payeepublicaddress":
		return "Payee Public Address"
	case "payerfioaddress":
		return "Payer FIO Address"
	case "payerkey":
		return "Payer Key"
	case "payerpublicaddress":
		return "Payer Public Address"
	case "status":
		return "Status"
	case "time":
		return "Time"
	case "timestamp":
		return "Time Stamp"
	case "tokencode":
		return "Token Code"
	}
	return s
}

func printTemplate() {
	fmt.Print("\n\nRequest Template:\n⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺\n\n")
	j, _ := json.MarshalIndent(&fio.ObtRequestContent{}, "", "  ")
	fmt.Println(`'` + string(j) + `'`)

	fmt.Print("\n\nRecord Response Template:\n⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺⎺\n\n")
	j, _ = json.MarshalIndent(&fio.ObtRecordContent{}, "", "  ")
	fmt.Println(`'` + string(j) + `'`)
	fmt.Print("\nMore information:\n  https://developers.fioprotocol.io/api/api-spec/models/fio-request-ecrypted-content\n  https://developers.fioprotocol.io/api/api-spec/models/fio-data-encrypted-content\n\n")
}

func printExample() {
	fmt.Println("\nImportant Options:")
	fmt.Println("------------------")
	fmt.Printf("  %9s 'URL for FIO nodeos endpoint'\n", "-u")
	fmt.Printf("  %9s 'password for keosd wallet'\n", "-password")
	fmt.Printf("  %9s 'fioreq command'\n", "-c")
	fmt.Printf("  %9s 'actor permission (account) for transaction'\n", "-p")
	fmt.Printf("  %9s 'request ID for command'\n", "-id")
	fmt.Printf("  %9s 'FIO Address that *recieves* funds'\n", "-payee")
	fmt.Printf("  %9s 'FIO Address that *sends* funds'\n", "-payer")
	fmt.Println("\n\nView available accounts from keosd:")
	fmt.Println("-----------------------------------")
	fmt.Println("  fioreq -u https://testnet.fioprotocol.io -password PW5xxxx.... -c list")
	fmt.Println("\n\nView sent requests for an account:")
	fmt.Println("----------------------------------")
	fmt.Println("  fioreq -u https://testnet.fioprotocol.io -password PW5xxxx.... -p aaaaaaaaaaaa -c sent")
	fmt.Println("\n\nView pending requests for an account:")
	fmt.Println("-------------------------------------")
	fmt.Println("  fioreq -u https://testnet.fioprotocol.io -password PW5xxxx.... -p aaaaaaaaaaaa -c pending")
	fmt.Println("\n\nView details for a request (including response):")
	fmt.Println("------------------------------------------------")
	fmt.Println("  fioreq -u https://testnet.fioprotocol.io -password PW5xxxx.... -p aaaaaaaaaaaa -c view-req -id 123")
	fmt.Println("\n\nReject a pending request")
	fmt.Println("------------------------")
	fmt.Println("  fioreq -u https://testnet.fioprotocol.io -password PW5xxxx.... -p aaaaaaaaaaaa -c reject -id 321")
	fmt.Println("\n\nRequest Payment:")
	fmt.Println("----------------")
	fmt.Println("  fioreq -u https://testnet.fioprotocol.io -password PW5xxxx.... -p aaaaaaaaaaaa -c request -payer shopper@fiotestnet -payee merchant@store '")
	fmt.Println(`    {
      "payee_public_address": "0x42F6cA7898A0f29e17CB66190f9E9B9d26f7D635",
      "amount": "123.45",
      "chain_code": "ETH",
      "token_code": "USDT",
      "memo": "payment for order 123"
    }'`)
	fmt.Println("\n\nRecord a transaction for a pending request")
	fmt.Println("------------------------------------------")
	fmt.Println("  fioreq -u https://testnet.fioprotocol.io -password PW5xxxx.... -p aaaaaaaaaaaa -c record -id 321 -payee merchant@store -payer shopper@fiotestnet '")
	fmt.Println(`    {
      "payer_public_address": "FIO6ZJ9p6ZSvboXqaFiowR8bKLtSk8ZGUTHdT8ZkaW6pNnbusPdwa",
      "payee_public_address": "FIO6QtJu52ho38zRP4aZCcgtciLAWQUB3CBgXnmwfFfXi6LvfVYyj",
      "amount": "1.000",
      "chain_code": "FIO",
      "token_code": "FIO",
      "hash": "797c59d1601f6bd99f0b56deb2c4fca944501a12b750829b66e4f792b0019fd4"
    }'`)
	fmt.Println("")
}

func ppYaml(title string, y []byte) {
	fmt.Printf("\n%19s:\n", title)
	fmt.Printf("%19s\n", strings.Repeat("⎺", len(title)))
	lines := strings.Split(string(y), "\n")
	for _, l := range lines {
		cols := strings.Split(l, ": ")
		if len(cols) == 2 && cols[1] != `""` {
			if len(cols[1]) > 64 {
				cols[1] = cols[1][:61] + "..."
			}
			cols[1] = strings.TrimPrefix(cols[1], `"`)
			cols[1] = strings.TrimSuffix(cols[1], `"`)
			fmt.Printf("%20s   %s\n", fixupFieldName(cols[0]), cols[1])
		}
	}
}

func rawJsonPrintLn(raw json.RawMessage) {
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
