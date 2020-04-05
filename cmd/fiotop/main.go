package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dapixio/fio-go"
	"github.com/eoscanada/eos-go"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"io/ioutil"
	"log"
	"sync"
	"time"
)

var Url string
var drawMux sync.Mutex

type logChanRecord struct {
	L *eos.Action
	T eos.Checksum256
	B uint32
}

func main() {

	var url = flag.String("u", "http://127.0.0.1:8888", "url to connect to.")
	flag.Parse()
	Url = *url

	var currentProducer eos.AccountName
	var highestBlock uint32
	var connectedPeers string
	var paused bool
	var showTx bool
	logChan := make(chan logChanRecord)

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

	var helpModal bool
	help := widgets.NewParagraph()
	help.TextStyle = ui.NewStyle(ui.ColorClear)

	// layout panes
	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	gridNormal := func() {
		grid.Set(
			ui.NewRow(1.0/8,
				ui.NewCol(1.0/2, p),
				ui.NewCol(1.0/2, g0),
			),
			ui.NewRow(
				0.9,
				ui.NewCol(0.3, prods),
				ui.NewCol(0.7,
					ui.NewRow(0.3,
						ui.NewCol(1.0, slg),
					),
					ui.NewRow(0.7,
						ui.NewCol(1.0, logs),
					),
				),
			),
		)
	}
	gridNormal()
	ui.Render(grid)

	// Chain info
	go func() {
		p.TextStyle.Fg = ui.ColorClear
		p.BorderStyle.Fg = ui.ColorClear
		for {
			info, e := api.GetInfo()
			if e != nil {
				if !helpModal {
					drawMux.Lock()
					p.TextStyle.Fg = ui.ColorRed
					p.Title = "Error"
					p.Text = e.Error()
					ui.Render(p)
					drawMux.Unlock()
				}
				time.Sleep(5 * time.Second)
			} else if !helpModal {
				currentProducer = info.HeadBlockProducer
				highestBlock = info.HeadBlockNum
				lag := info.HeadBlockTime.Sub(time.Now().UTC()) / time.Second
				drawMux.Lock()
				p.Title = fmt.Sprintf("nodeos: %s @ %s", info.ServerVersionString, Url)
				p.TextStyle.Fg = ui.ColorClear
				p.Text = pr.Sprintf(
					"\n    Head: %d  Irreversible: %d\n    %s",
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
				drawMux.Unlock()
			}
			time.Sleep(250 * time.Millisecond)
		}
	}()

	// db size info
	go func() {
		g0.BorderStyle.Fg = ui.ColorClear
		g0.TitleStyle.Fg = ui.ColorClear
		for {

			size, e := api.GetDBSize()
			if e != nil {
				if !helpModal {
					g0.Title = " get db size failed, is db_size_api_plugin enabled? "
					g0.TitleStyle.Fg = ui.ColorYellow
					g0.BarColor = ui.ColorRed
					g0.Percent = 0
					drawMux.Lock()
					ui.Render(g0)
					drawMux.Unlock()
				}
				time.Sleep(10 * time.Second)
			} else if !helpModal {
				drawMux.Lock()
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
				ui.Render(g0)
				drawMux.Unlock()
			}
			time.Sleep(3 * time.Second)
		}
	}()

	// Producer info
	lastProduced := make(map[eos.AccountName]time.Time)
	go func() {
		prods.Title = "Current Producers"
		prods.TextStyle = ui.StyleClear
		for {
			bps := make([][]string, 0)
			styles := make(map[int]ui.Style)
			gfp, err := api.GetFioProducers()
			if err != nil {
				prods.Title = "Cannot get list."
				prods.TextStyle = ui.NewStyle(ui.ColorRed)
				prods.BorderStyle = ui.NewStyle(ui.ColorClear)
				if !helpModal {
					drawMux.Lock()
					ui.Render(prods)
					drawMux.Unlock()
				}
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
				//styles[0] = ui.NewStyle(ui.ColorCyan)
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
							//styles[i+2] = ui.NewStyle(ui.ColorGreen, ui.ColorBlack, ui.ModifierUnderline)
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
				if !helpModal {
					drawMux.Lock()
					prods.Rows = bps
					prods.RowStyles = styles
					prods.TextAlignment = ui.AlignRight
					prods.RowSeparator = false
					prods.FillRow = false
					ui.Render(prods)
					drawMux.Unlock()
				}
			}
			time.Sleep(time.Second)
		}
	}()

	// Tx sparkline
	ticks := slg.Size()
	txCount := make([]float64, ticks.X-2)
	for i := range txCount {
		txCount[i] = 0.0
	}
	go func() {
		if !helpModal {
			drawMux.Lock()
			sl.Title = "TX / Block"
			sl.LineColor = ui.ColorBlue
			sl.Data = txCount
			ui.Render(slg)
			drawMux.Unlock()
		}
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
				for paused {
					time.Sleep(time.Second)
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
								logChan <- logChanRecord{L: a, T: tx.Transaction.ID, B: b.BlockNum}
							}
						}
					}
				}
			}()
			var count int
			var title string
			// tpb average display for last 10 blocks
			if len(txCount) >= 10 {
				for _, i := range txCount[len(txCount)-30:] {
					count = count + int(i)
				}
				blocks := 10.0
				avg := float64(count) / blocks / 2.0
				title = pr.Sprintf("TX / Block (avg %.2f/block)", avg)
			}
			if !helpModal {
				drawMux.Lock()
				sl.Title = title
				sl.Data = txCount
				ui.Render(slg)
				drawMux.Unlock()
			}
			time.Sleep(time.Second)
		}
	}()

	// update peer info
	go func() {
		for {
			n, e := getPeerCounts(api)
			if e != nil {
				time.Sleep(10 * time.Minute) // try again later
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
			drawMux.Lock()
			connectedPeers = pr.Sprintf("Peers: %d total (%d inbound connections, %d syncing)", total, inbound, syncing)
			drawMux.Unlock()
			time.Sleep(10 * time.Second)
		}
	}()

	// action stream
	logBuffer := make([]string, 80)
	go func() {
		for paused {
			time.Sleep(time.Second)
		}
		lpushRPop := func(l string) {
			logBuffer = append([]string{l}, logBuffer[:len(logBuffer)-1]...)
		}
		var abis map[eos.AccountName]*eos.ABI
		// spin until we have the abi's loaded ...
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
				if abis[l.L.Account] != nil {
					var e error
					actionData, e = abis[l.L.Account].DecodeTableRowTyped(string(l.L.Name), l.L.HexData)
					if e != nil {
						actionData = []byte(hexutil.Encode(l.L.HexData))
					}
				}
				var line string
				switch showTx {
				case false:
					line = pr.Sprintf("%s %s %s::%s -- %s", time.Now().Format("15:04:05.000"), l.L.Authorization[0].Actor, l.L.Account, l.L.Name, string(actionData))
				case true:
					line = pr.Sprintf("%s (%v) %s:\n             %s %s::%s -- %s", time.Now().Format("15:04:05.000"), l.B, l.T.String(), l.L.Authorization[0].Actor, l.L.Account, l.L.Name, string(actionData))
				}
				lpushRPop(line)
				if !helpModal {
					drawMux.Lock()
					logs.Rows = logBuffer
					ui.Render(logs)
					drawMux.Unlock()
				}
			}
		}
	}()

	// repaint screen
	repaint := func() {
		if !helpModal {
			drawMux.Lock()
			ui.Clear()
			termWidth, termHeight = ui.TerminalDimensions()
			grid.SetRect(0, 0, termWidth, termHeight)
			ui.Render(grid)
			drawMux.Unlock()
		}
	}

	go func() {
		for {
			time.Sleep(30 * time.Second)
			repaint()
		}
	}()

	gridHelp := func() {
		grid.Set(
			ui.NewRow(1.0/3,
				ui.NewCol(1.0/2, help),
			),
		)
	}

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			return
		case "<Resize>":
			if !helpModal {
				drawMux.Lock()
				payload := e.Payload.(ui.Resize)
				ui.Clear()
				grid.SetRect(0, 0, payload.Width, payload.Height)
				ui.Render(grid)
				drawMux.Unlock()
			}
		// allow user to request repaint
		case "r", "<C-l>":
			repaint()
		// wipeout clears data
		case "d", "<C-u>":
			if !helpModal {
				drawMux.Lock()
				connectedPeers = ""
				logBuffer = make([]string, 80)
				logs.Rows = logBuffer
				txCount = make([]float64, ticks.X-2)
				for i := range txCount {
					txCount[i] = 0.0
				}
				lastProduced = make(map[eos.AccountName]time.Time)
				g0.Title = ""
				ui.Render(g0, logs, slg, prods, p)
				drawMux.Unlock()
			}
		case "t":
			switch showTx {
			case true:
				showTx = false
			case false:
				showTx = true
			}
		case "p":
			switch paused {
			case true:
				paused = false
			case false:
				paused = true
			}
		// help modal
		case "?", "<F1>":
			drawMux.Lock()
			helpModal = true
			help.Title = " Help "
			help.Text = helpText
			help.Border = true
			gridHelp()
			ui.Render(grid)
			ui.Clear()
			ui.Render(help)
			drawMux.Unlock()
		// clear help modal
		case "<Escape>", "<Enter>":
			if helpModal {
				drawMux.Lock()
				help.Title = ""
				help.Text = ""
				help.Border = false
				helpModal = false
				ui.Clear()
				gridNormal()
				ui.Render(grid)
				drawMux.Unlock()
			}
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

const helpText = `
 Keys:

    q or CTRL-C to exit
    r or CTRL-L to repaint screen
    d or CTRL-U to clear data
    p to pause event stream
    t to show txid in event stream

    Press ESC or ENTER to exit help.
`
