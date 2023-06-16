package main

import "os/exec"

type ChainData struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Regtest     bool   `json:"regtest"`
	Bin         string `json:"bin"`
	Port        string `json:"port"`
	RPCUser     string `json:"rpc_user"`
	RPCPass     string `json:"rpc_password"`
}

type ChainState struct {
	ID         string `json:"id"`
	State      State  `json:"state"`
	RefreshBMM bool   `json:"refresh_bmm"`
	CMD        *exec.Cmd
}

type RPCRequest struct {
	JSONRpc string   `json:"jsonrpc"`
	ID      string   `json:"id"`
	Method  string   `json:"method"`
	Params  []string `json:"params"`
}

type State uint

const (
	Unknown State = iota
	Waiting
	Running
)
