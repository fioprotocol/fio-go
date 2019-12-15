package main

import (
	"encoding/json"
	"flag"
	"github.com/dapixio/fio-go"
	"github.com/eoscanada/eos-go"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"io/ioutil"
	"log"
	"time"
)

var Url string

func main() {

	var url = flag.String("u", "http://127.0.0.1:8888", "url to connect to.")
	flag.Parse()
	Url = *url

	var currentProducer eos.AccountName
	var highestBlock uint32
	var connectedPeers string
	logChan := make(chan *eos.Action)
	api, _, err := fio.NewConnection(nil, Url)
	if err != nil {
		log.Fatal(err)
	}
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	// setup our panes
	pr := message.NewPrinter(language.AmericanEnglish)
	p := widgets.NewParagraph()
	prods := widgets.NewTable()
	prods.Rows = [][]string{{""}}
	g0 := widgets.NewGauge()
	sl := widgets.NewSparkline()
	slg := widgets.NewSparklineGroup(sl)
	logs := widgets.NewList()

	// layout panes
	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	grid.Set(
		ui.NewRow(1.0/8,
			ui.NewCol(1.0/2, p),
			ui.NewCol(1.0/2, g0),
		),
		ui.NewRow(
			0.85,
			ui.NewCol(0.28, prods),
			ui.NewCol(0.72,
				ui.NewRow(0.3,
					ui.NewCol(1.0, slg),
				),
				ui.NewRow(0.7,
					ui.NewCol(1.0, logs),
				),
			),
		),
	)
	ui.Render(grid)

	// Chain info
	go func() {
		p.TextStyle.Fg = ui.ColorClear
		p.BorderStyle.Fg = ui.ColorClear
		for {
			//p.SetRect(0, 0, halfWidth - 4, quarterHeight - 3)
			info, e := api.GetInfo()
			if e != nil {
				p.TextStyle.Fg = ui.ColorRed
				p.Title = "Error"
				p.Text = e.Error()
				ui.Render(p)
				time.Sleep(5 * time.Second)
			} else {
				currentProducer = info.HeadBlockProducer
				highestBlock = info.HeadBlockNum
				p.Title = "nodeos: " + info.ServerVersionString
				lag := info.HeadBlockTime.Sub(time.Now().UTC()) / time.Second
				p.TextStyle.Fg = ui.ColorClear
				p.Text = pr.Sprintf(
					"\n    Head: %d  Irr: %d\n    %s",
					info.HeadBlockNum, info.LastIrreversibleBlockNum, connectedPeers,
				)
				if lag > 0 {
					p.TextStyle.Fg = ui.ColorYellow
					p.Text = pr.Sprintf(
						"\n    Head: %d  Irr: %d Lag (s): %d\n    %s",
						info.HeadBlockNum, info.LastIrreversibleBlockNum, lag, connectedPeers,
					)
				}

				ui.Render(p)
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	// db size info
	go func() {
		g0.BorderStyle.Fg = ui.ColorClear
		g0.TitleStyle.Fg = ui.ColorClear
		for {
			//g0.SetRect(halfWidth + 6, 0, halfWidth - 11, quarterHeight - 3)
			size, e := api.GetDBSize()
			if e != nil {
				g0.Title = " get db size failed "
				g0.TitleStyle.Fg = ui.ColorYellow
				g0.BarColor = ui.ColorRed
				g0.Percent = 0
				ui.Render(g0)
				time.Sleep(5 * time.Second)
			} else {
				g0.TitleStyle.Fg = ui.ColorClear
				g0.Title = pr.Sprintf("Database (mem) %d / %d MiB", size.UsedBytes/(1024*1024), size.Size/(1024*1024))
				pct := int(100 - 100.0*(float32(size.FreeBytes)/float32(size.Size)))
				g0.Percent = pct
				switch {
				case pct < 50:
					g0.BarColor = ui.ColorGreen
				case pct >= 50 && pct < 75:
					g0.TitleStyle.Fg = ui.ColorYellow
					g0.BarColor = ui.ColorYellow
				case pct >= 75:
					g0.TitleStyle.Fg = ui.ColorRed
					g0.BarColor = ui.ColorRed
				}
			}
			ui.Render(g0)
			time.Sleep(3 * time.Second)
		}
	}()

	// Producer info
	go func() {
		prods.Title = "Current Producers"
		prods.TextStyle = ui.StyleClear
		lastProduced := make(map[eos.AccountName]time.Time)
		for {
			bps := make([][]string, 0)
			styles := make(map[int]ui.Style)
			gfp, err := api.GetFioProducers()
			if err != nil {
				prods.Title = "Cannot get list."
				prods.TextStyle = ui.NewStyle(ui.ColorRed)
				prods.BorderStyle = ui.NewStyle(ui.ColorClear)
				ui.Render(prods)
				time.Sleep(5 * time.Second)
			} else {
				bps = append(bps,
					[]string{
						"Producer",
						"Last Produced",
					},
					[]string{
						"",
						"",
					},
				)
				styles[0] = ui.NewStyle(ui.ColorCyan)
				styles[0] = ui.NewStyle(ui.ColorClear)
				for i, p := range gfp.Producers {
					if p.IsActive > 0 {
						last := "--"
						if lastProduced[p.Owner].Unix() > 0 {
							dur := lastProduced[p.Owner].Sub(time.Now()).Seconds()
							last = (time.Second*time.Duration(dur) + time.Second).String()
						}
						bps = append(bps,
							[]string{
								string(p.FioAddress),
								last,
							},
						)
						switch {
						case currentProducer == p.Owner:
							styles[i+2] = ui.NewStyle(ui.ColorGreen)
							lastProduced[p.Owner] = time.Now()
						case i >= 21:
							styles[i+2] = ui.NewStyle(ui.ColorBlue)
						case lastProduced[p.Owner].Unix() > 0 && lastProduced[p.Owner].Before(time.Now().Add(-122*time.Second)):
							styles[i+2] = ui.NewStyle(ui.ColorYellow)
						default:
							styles[i+2] = ui.NewStyle(ui.ColorClear)
						}

					}
				}
				prods.Rows = bps
				prods.RowStyles = styles
				prods.TextAlignment = ui.AlignRight
				prods.RowSeparator = false
				prods.FillRow = false
				ui.Render(prods)
			}
			time.Sleep(time.Second)
		}
	}()

	// Tx sparkline
	go func() {
		sl.Title = "TX / Block"
		sl.LineColor = ui.ColorBlue
		ticks := slg.Size()
		txCount := make([]float64, ticks.X-2)
		for i := range txCount {
			txCount[i] = 0.0
		}
		sl.Data = txCount
		ui.Render(slg)
		pushPop := func(last float64) {
			txCount = append(txCount[1:], last)
		}
		current := highestBlock
		for {
			func() {
				if current == 0 {
					current = highestBlock
					return
				}
				next := highestBlock
				if next > current && current > 0 {
					for i := 1; i <= int(next-current); i++ {
						b, err := api.GetBlockByNum(current + uint32(i))
						if err != nil {
							return
						}
						pushPop(float64(len(b.Transactions)))
						current = current + uint32(i)
						for _, tx := range b.Transactions {
							s, e := tx.Transaction.Packed.Unpack()
							if e != nil {
								continue
							}
							for _, a := range s.Actions {
								logChan <- a
							}
						}
					}
				}
			}()
			sl.Data = txCount
			ui.Render(slg)
			time.Sleep(time.Second)
		}
	}()

	// update peer info
	go func() {
		for {
			n, e := getPeerCounts(api)
			if e != nil {
				connectedPeers = "Cannot get peer information, is net_api_plugin enabled?"
				//connectedPeers = e.Error()
				time.Sleep(10 * time.Minute)
				continue
			}
			var (
				total   = len(n)
				inbound int
				syncing int
			)
			for _, p := range n {
				if p.Syncing {
					syncing = syncing + 1
				}
				if p.Connecting {
					inbound = inbound + 1
				}
			}
			connectedPeers = pr.Sprintf("Peers: %d total (%d inbound connections, %d syncing)", total, inbound, syncing)
			time.Sleep(10 * time.Second)
		}
	}()

	// action stream
	go func() {
		//size := logs.Size()
		logBuffer := make([]string, 80)
		lpushRPop := func(l string) {
			logBuffer = append([]string{l}, logBuffer[:len(logBuffer)-1]...)
		}
		var abis map[eos.AccountName]*eos.ABI
		for {
			var e error
			abis, e = api.AllABIs()
			if e == nil && len(abis) > 0 {
				break
			}
			time.Sleep(3 * time.Second)
		}
		for {
			for l := range logChan {
				actionData := make([]byte, 0)
				if abis[l.Account] != nil {
					var e error
					actionData, e = abis[l.Account].DecodeTableRowTyped(string(l.Name), l.HexData)
					if e != nil {
						actionData = []byte(hexutil.Encode(l.HexData))
					}
				}
				line := pr.Sprintf("%s %s %s::%s -- %s", time.Now().Format("15:04:05.000"), l.Authorization[0].Actor, l.Account, l.Name, string(actionData))
				lpushRPop(line)
				logs.Rows = logBuffer
				ui.Render(logs)
			}
		}
	}()

	// repaint screen
	go func() {
		for {
			time.Sleep(30 * time.Second)
			ui.Clear()
			termWidth, termHeight = ui.TerminalDimensions()
			grid.SetRect(0, 0, termWidth, termHeight)
			ui.Render(grid)
		}
	}()

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			return
		case "<Resize>":
			payload := e.Payload.(ui.Resize)
			grid.SetRect(0, 0, payload.Width, payload.Height)
			ui.Clear()
			ui.Render(grid)
		}
	}

}

type peers struct {
	Connecting bool `json:"connecting"`
	Syncing    bool `json:"syncing"`
}

// TODO: make a FIO compatible call available ....
func getPeerCounts(api *fio.API) ([]peers, error) {
	resp, err := api.HttpClient.Post(Url+"/v1/net/connections", "application/json", nil)
	if err != nil {
		return []peers{}, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []peers{}, err
	}
	peerList := make([]peers, 0)
	err = json.Unmarshal(body, &peerList)
	if err != nil {
		return []peers{}, err
	}
	return peerList, nil
}

