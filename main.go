package main

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	ps "github.com/mitchellh/go-ps"
)

const APPID = "com.layertwolabs.switchboard"
const APP_TITLE = "DriveNet Switchboard"

var baseSize = fyne.Size{Width: 900, Height: 640}

var a fyne.App
var w fyne.Window

var content = container.NewVBox()

//go:embed binaries/drivechain-qt-darwin
var drivechainDarwinQtBytes []byte

//go:embed binaries/drivechain-qt-linux
var drivechainLinuxQtBytes []byte

//go:embed binaries/drivechain-qt-windows.exe
var drivechainWindowsQtBytes []byte

//go:embed binaries/bitassets-qt-darwin
var bitassetsDarwinQtBytes []byte

//go:embed binaries/bitassets-qt-linux
var bitassetsLinuxQtBytes []byte

//go:embed binaries/bitassets-qt-windows.exe
var bitassetsWindowsQtBytes []byte

//go:embed binaries/testchain-qt-darwin
var testchainDarwinQtBytes []byte

//go:embed binaries/testchain-qt-linux
var testchainLinuxQtBytes []byte

//go:embed binaries/testchain-qt-windows.exe
var testchainWindowsQtBytes []byte

//go:embed images/mine.svg
var mineIconBytes []byte
var mineIcon fyne.Resource

// //go:embed images/start.svg
// var startIconBytes []byte
// var startIcon fyne.Resource

//go:embed images/stop.png
var stopIconBytes []byte
var stopIcon fyne.Resource

//go:embed data/chain_data.json
var chainDataBytes []byte

var chainStateUpdate *time.Ticker
var quitChainStateUpdate chan struct{}

var chainData = make([]ChainData, 0)
var chainState = make(map[string]ChainState, 0)

var selectedChainDataIndex int = 0
var switchboardDir string

func main() {
	a = app.NewWithID(APPID)
	w = a.NewWindow(APP_TITLE)
	w.Resize(baseSize)

	dirSetup()

	mineIcon = fyne.NewStaticResource("mine.svg", mineIconBytes)
	//startIcon = fyne.NewStaticResource("start.svg", startIconBytes)
	stopIcon = fyne.NewStaticResource("stop.png", stopIconBytes)

	// UI Setup
	// Create the left menu
	var leftMenuWidgetList *widget.List
	leftMenuWidgetList = widget.NewList(
		func() int {
			return len(chainData)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(chainData[i].Name)
		},
	)
	leftMenuWidgetList.OnSelected = func(id widget.ListItemID) {
		selectedChainDataIndex = id
		setMainContentUI(selectedChainDataIndex)
	}
	leftMenuWidgetList.Select(0)

	leftBorderContainer := container.NewBorder(nil, nil, nil, nil, leftMenuWidgetList)
	rightBorderContainer := container.NewBorder(nil, nil, nil, nil, content)

	splitContainer := container.NewHSplit(leftBorderContainer, rightBorderContainer)
	splitContainer.Offset = 0.3

	w.SetContent(splitContainer)

	startChainStateUpdate()

	w.ShowAndRun()

	cleanup()
}

func dirSetup() {
	// Load Chain Data. This will eventually be pulled from a remote server
	err := json.Unmarshal(chainDataBytes, &chainData)
	if err != nil {
		log.Fatal(err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	if _, err := os.Stat(homeDir + "/.switchboard3"); os.IsNotExist(err) {
		os.Mkdir(homeDir+"/.switchboard3", 0755)
	}

	switchboardDir = homeDir + "/.switchboard3"

	writeBinaries()
}

func writeBinaries() {
	for i, chain := range chainData {
		if i > 0 {
			target := runtime.GOOS
			switch target {
			case "darwin":
				switch chain.ID {
				case "drivechain":
					err := os.WriteFile(switchboardDir+"/"+chain.Bin, drivechainDarwinQtBytes, 0755)
					if err != nil {
						log.Fatal(err)
					}
				case "bitassets":
					err := os.WriteFile(switchboardDir+"/"+chain.Bin, bitassetsDarwinQtBytes, 0755)
					if err != nil {
						log.Fatal(err)
					}
				case "testchain":
					err := os.WriteFile(switchboardDir+"/"+chain.Bin, testchainDarwinQtBytes, 0755)
					if err != nil {
						log.Fatal(err)
					}
				}
			case "linux":
				switch chain.ID {
				case "drivechain":
					err := os.WriteFile(switchboardDir+"/"+chain.Bin, drivechainLinuxQtBytes, 0755)
					if err != nil {
						log.Fatal(err)
					}
				case "bitassets":
					err := os.WriteFile(switchboardDir+"/"+chain.Bin, bitassetsLinuxQtBytes, 0755)
					if err != nil {
						log.Fatal(err)
					}
				case "testchain":
					err := os.WriteFile(switchboardDir+"/"+chain.Bin, testchainLinuxQtBytes, 0755)
					if err != nil {
						log.Fatal(err)
					}
				}
			case "windows":
				switch chain.ID {
				case "drivechain":
					err := os.WriteFile(switchboardDir+"/"+chain.Bin, drivechainWindowsQtBytes, 0755)
					if err != nil {
						log.Fatal(err)
					}
				case "bitassets":
					err := os.WriteFile(switchboardDir+"/"+chain.Bin, bitassetsWindowsQtBytes, 0755)
					if err != nil {
						log.Fatal(err)
					}
				case "testchain":
					err := os.WriteFile(switchboardDir+"/"+chain.Bin, testchainWindowsQtBytes, 0755)
					if err != nil {
						log.Fatal(err)
					}
				}
			}
		}
	}
}

func launchChain(chainDataIndex int) {
	chain := chainData[chainDataIndex]
	chainDataDir := switchboardDir + "/data/" + chain.ID
	if _, err := os.Stat(chainDataDir); os.IsNotExist(err) {
		os.MkdirAll(chainDataDir, 0755)
	}
	var regtest string = "0"
	if chain.Regtest {
		regtest = "1"
	}
	args := []string{"-regtest=" + regtest, "-datadir=" + chainDataDir, "-rpcport=" + chain.Port, "-rpcuser=" + chain.RPCUser, "-rpcpassword=" + chain.RPCPass, "-server=1"}
	cmd := exec.Command(switchboardDir+"/"+chain.Bin, args...)
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	chainState[chain.ID] = ChainState{ID: chain.ID, State: Waiting, CMD: cmd}

	setMainContentUI(selectedChainDataIndex)
}

func stopChain(chainDataIndex int) {
	chain := chainData[chainDataIndex]
	v, ok := chainState[chain.ID]
	if ok {
		err := v.CMD.Process.Kill()
		if err != nil {
			log.Fatal(err)
		}
		delete(chainState, chain.ID)
		setMainContentUI(selectedChainDataIndex)
	}
}

func makeRpcRequest(chainDataIndex int, method string, params []string) (*http.Response, error) {
	if chainDataIndex > len(chainData) {
		return nil, errors.New("invalid chainDataIndex")
	}
	chain := chainData[chainDataIndex]
	auth := chain.RPCUser + ":" + chain.RPCPass
	authBytes := []byte(auth)
	authEncoded := base64.StdEncoding.EncodeToString(authBytes)
	rpcRequest := RPCRequest{JSONRpc: "2.0", ID: "switchboard", Method: method, Params: params}
	body, err := json.Marshal(rpcRequest)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	req, err := http.NewRequest("POST", "http://localhost:"+chain.Port, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Basic "+authEncoded)
	req.Header.Add("content-type", "application/json")
	return client.Do(req)
}

func startChainStateUpdate() {
	chainStateUpdate = time.NewTicker(1 * time.Second)
	quitChainStateUpdate = make(chan struct{})
	go func() {
		for {
			select {
			case <-chainStateUpdate.C:
				// call RPC method getblockcount to check if chain is running
				for k, state := range chainState {
					// This is used to handle case where user stops chain from outside of switchboard
					if !isProcessRunning(state.CMD) {
						delete(chainState, k)
						setMainContentUI(selectedChainDataIndex)
					} else {
						resp, err := makeRpcRequest(getChainDataIndexByID(k), "getblockcount", []string{})
						if err != nil {
							if state.State == Running {
								state.State = Waiting
								chainState[k] = state
								setMainContentUI(selectedChainDataIndex)
							}
						} else {

							defer resp.Body.Close()
							if resp.StatusCode == 200 {
								if state.State == Waiting || state.State == Unknown {
									state.State = Running
									chainState[k] = state
									setMainContentUI(selectedChainDataIndex)
								}
							}
						}
					}
				}
			case <-quitChainStateUpdate:
				chainStateUpdate.Stop()
				return
			}
		}
	}()
}

func isProcessRunning(cmd *exec.Cmd) bool {
	pid := cmd.Process.Pid
	process, err := ps.FindProcess(pid)
	return err == nil && process != nil // TODO: Bug, for some reason process will be found on second run.. Even with different pid??
}

func getChainDataIndexByID(id string) int {
	for i, chain := range chainData {
		if chain.ID == id {
			return i
		}
	}
	return -1
}

func cleanup() {
	for _, c := range chainState {
		err := c.CMD.Process.Kill()
		if err != nil {
			log.Fatal(err)
		}
	}
	quitChainStateUpdate <- struct{}{}
}

func cardButton(label string, disabled bool, tapped func()) *widget.Button {
	b := widget.NewButton(label, tapped)
	if disabled {
		b.Disable()
	}
	return b
}

func chainCard(chainData ChainData, state *ChainState, mainChainState *ChainState) *widget.Card {

	chainIndex := getChainDataIndexByID(chainData.ID)
	st := widget.NewLabel(chainData.Description)
	st.Wrapping = fyne.TextWrapWord

	var launchButton *widget.Button
	cardContainer := container.NewVBox(st, layout.NewSpacer())

	if chainData.ID == "drivechain" {

		if mainChainState.ID == "" {
			launchButton = cardButton("Launch", false, func() {
				launchChain(1)
			})
			launchButton.Importance = widget.HighImportance
			cardContainer.Add(launchButton)
		} else if mainChainState.ID != "" && mainChainState.State == Waiting {
			launchButton = cardButton("Starting...", true, func() {})
			cardContainer.Add(launchButton)
		} else if mainChainState.ID != "" && mainChainState.State == Running {
			launchButton = cardButton("Stop", false, func() {
				stopChain(1)
			})
			mineButton := widget.NewButton("Mine", func() {
			})
			mineButton.Importance = widget.HighImportance
			cardContainer.Add(container.NewVBox(launchButton, mineButton))
		}

	} else {

		if state.ID == "" && mainChainState.ID != "" && mainChainState.State == Running {
			launchButton = cardButton("Launch", false, func() {
				launchChain(chainIndex)
			})
			launchButton.Importance = widget.HighImportance
			cardContainer.Add(launchButton)
		} else if state.ID != "" && state.State == Waiting {
			launchButton = cardButton("Starting...", true, func() {})
			cardContainer.Add(launchButton)
		} else if state.ID != "" && state.State == Running {
			launchButton = cardButton("Stop", false, func() {
				stopChain(chainIndex)
			})
			cardContainer.Add(launchButton)
		} else {
			launchButton = cardButton("Launch", true, func() {})
			launchButton.Importance = widget.HighImportance
			cardContainer.Add(launchButton)
		}

	}

	return widget.NewCard(chainData.Name, "", cardContainer)
}

func setMainContentUI(chainDataIndex int) {
	mainChain := chainData[1]
	chain := chainData[chainDataIndex]

	content.Objects = nil

	mainChainState := chainState[mainChain.ID]

	if chain.ID == "overview" {

		// Top area for main chain
		vbox := container.NewVBox()
		vbox.Add(chainCard(mainChain, &mainChainState, &mainChainState))

		// Bottom area for sidechains
		grid := container.NewGridWithColumns(2)
		vbox.Add(grid)

		for i, chain := range chainData {
			if i > 1 {
				state := chainState[chain.ID]
				grid.Add(chainCard(chain, &state, &mainChainState))
			}
		}

		content.Add(container.NewPadded(vbox))

	} else {
		state := chainState[chain.ID]
		vbox := container.NewVBox()
		vbox.Add(chainCard(chain, &state, &mainChainState))
		content.Add(container.NewPadded(vbox))
	}

	content.Refresh()
}
