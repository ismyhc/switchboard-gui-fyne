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
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	ps "github.com/mitchellh/go-ps"
)

const APPID = "com.layertwolabs.switchboard"
const APP_TITLE = "DriveNet Switchboard"

var baseSize = fyne.Size{Width: 900, Height: 640}

var a fyne.App
var w fyne.Window
var t switchboardTheme

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

	t = switchboardTheme{}
	t.Init()

	a.Settings().SetTheme((fyne.Theme)(&t))
	w = a.NewWindow(APP_TITLE)
	w.Resize(baseSize)

	dirSetup()

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

	chainState[chain.ID] = ChainState{ID: chain.ID, State: Waiting, RefreshBMM: true, CMD: cmd}

	setMainContentUI(selectedChainDataIndex)
}

func toggleRefreshBMM(chainID string) {
	v, ok := chainState[chainID]
	if ok {
		v.RefreshBMM = !v.RefreshBMM
		chainState[chainID] = v
	}
}

func stopChain(chainDataIndex int) {
	chain := chainData[chainDataIndex]
	_, ok := chainState[chain.ID]
	if ok {
		_, _ = makeRpcRequest(chainDataIndex, "stop", []interface{}{})
		delete(chainState, chain.ID)
		setMainContentUI(selectedChainDataIndex)
	}
}

func deposit(chainDataIndex int, amount float64) {
	chain := chainData[chainDataIndex]
	res, err := makeRpcRequest(getChainDataIndexByID(chain.ID), "getdepositaddress", []interface{}{})
	if err != nil {
		println(err.Error())
		return
	}
	defer res.Body.Close()
	if res.StatusCode == 200 {
		var rpcRes RPCGetDepositAddressResponse
		err := json.NewDecoder(res.Body).Decode(&rpcRes)
		if err == nil {
			address := rpcRes.Result
			_, err := makeRpcRequest(1, "createsidechaindeposit", []interface{}{chain.Slot, address, amount, 0.001})
			if err != nil {
				println(err.Error())
			}
		}
	}
}

func mainChainMine() {
	_, err := makeRpcRequest(1, "generate", []interface{}{1})
	if err != nil {
		println(err.Error())
	}
}

func makeRpcRequest(chainDataIndex int, method string, params []interface{}) (*http.Response, error) {
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
						resp, err := makeRpcRequest(getChainDataIndexByID(k), "getblockcount", []interface{}{})
						if err != nil {
							if state.State == Running {
								state.State = Waiting
								chainState[k] = state
								setMainContentUI(selectedChainDataIndex)
							}
						} else {
							defer resp.Body.Close()
							if resp.StatusCode == 200 {

								if state.ID == "drivechain" {
									var res RPCGetBlockCountResponse
									err := json.NewDecoder(resp.Body).Decode(&res)
									if err == nil {
										if res.Result < 200 {
											// loop chain data
											for _, chain := range chainData {
												if chain.ID != "drivechain" && chain.ID != "overview" {
													_, err := makeRpcRequest(getChainDataIndexByID(state.ID), "createsidechainproposal", []interface{}{chain.Slot, chain.ID})
													if err != nil {
														println(err.Error())
													}
												}
											}
											_, _ = makeRpcRequest(getChainDataIndexByID(k), "generate", []interface{}{201})
										}
									}
								}

								if state.ID != "drivechain" && state.RefreshBMM {
									_, _ = makeRpcRequest(getChainDataIndexByID(k), "refreshbmm", []interface{}{0.001})
								}

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

	vers := canvas.NewText("  Version: "+chainData.Version, theme.ForegroundColor())
	vers.TextSize = theme.CaptionTextSize()
	vers.TextStyle = fyne.TextStyle{Italic: true, Bold: true}
	vers.Alignment = fyne.TextAlignLeading

	var launchButton *widget.Button
	cardContainer := container.NewVBox(st, container.NewPadded(vers), layout.NewSpacer())

	if chainData.ID == "drivechain" {

		if mainChainState.ID == "" {
			launchButton = cardButton(" Launch", false, func() {
				launchChain(1)
			})
			launchButton.SetIcon(t.Icon(StartIcon))
			launchButton.Importance = widget.HighImportance
			cardContainer.Add(launchButton)
		} else if mainChainState.ID != "" && mainChainState.State == Waiting {
			launchButton = cardButton(" Starting...", true, func() {})
			cardContainer.Add(launchButton)
		} else if mainChainState.ID != "" && mainChainState.State == Running {
			launchButton = cardButton(" Stop", false, func() {
				stopChain(1)
			})
			launchButton.SetIcon(t.Icon(StopIcon))
			mineButton := widget.NewButton(" Mine", func() {
				mainChainMine()
			})
			mineButton.SetIcon(t.Icon(MineIcon))
			mineButton.Importance = widget.HighImportance
			cardContainer.Add(container.NewVBox(launchButton, mineButton))
		}

	} else {

		refreshCheck := widget.NewCheck("Mine", func(b bool) {
			toggleRefreshBMM(chainData.ID)
		})
		cardContainer.Add(refreshCheck)

		depositEntry := widget.NewEntry()
		depositEntry.SetPlaceHolder("")
		depositEntry.Disable()

		depositButton := widget.NewButtonWithIcon("Deposit", t.Icon(DepositIcon), func() {
			value, err := strconv.ParseFloat(depositEntry.Text, 64)
			if err != nil {
				println(err.Error())
			}
			deposit(chainIndex, value)
		})
		depositButton.Disable()

		depositBorder := container.NewBorder(nil, nil, nil, depositButton, depositEntry)

		withdrawEntry := widget.NewEntry()
		withdrawEntry.SetPlaceHolder("")
		withdrawEntry.Disable()

		withdrawButton := widget.NewButtonWithIcon("Withdraw", t.Icon(WithdrawIcon), func() {
			// value, err := strconv.ParseFloat(depositEntry.Text, 64)
			// if err != nil {
			// 	println(err.Error())
			// }
			// deposit(chainIndex, value)
		})
		withdrawButton.Disable()

		withdrawBorder := container.NewBorder(nil, nil, nil, withdrawButton, withdrawEntry)

		cardContainer.Add(depositBorder)
		cardContainer.Add(withdrawBorder)

		if state.ID == "" && mainChainState.ID != "" && mainChainState.State == Running {
			refreshCheck.Checked = true
			refreshCheck.Refresh()
			refreshCheck.Disable()

			launchButton = cardButton(" Launch", false, func() {
				launchChain(chainIndex)
			})
			launchButton.SetIcon(t.Icon(StartIcon))
			launchButton.Importance = widget.HighImportance
			cardContainer.Add(launchButton)
		} else if state.ID != "" && state.State == Waiting {
			refreshCheck.Checked = state.RefreshBMM
			refreshCheck.Refresh()
			refreshCheck.Disable()
			launchButton = cardButton(" Starting...", true, func() {})
			cardContainer.Add(launchButton)
		} else if state.ID != "" && state.State == Running {
			refreshCheck.Checked = state.RefreshBMM
			refreshCheck.Refresh()

			depositEntry.Enable()
			depositButton.Enable()
			withdrawEntry.Enable()
			withdrawButton.Enable()

			launchButton = cardButton(" Stop", false, func() {
				stopChain(chainIndex)
			})
			launchButton.SetIcon(t.Icon(StopIcon))
			cardContainer.Add(launchButton)
		} else {
			refreshCheck.Checked = true
			refreshCheck.Refresh()
			refreshCheck.Disable()
			launchButton = cardButton(" Launch", true, func() {})
			launchButton.SetIcon(t.Icon(StartIcon))
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
