package main

import (
	"encoding/json"
	"io"
	"os"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func hostJSON() {
	response, code := processHostJSON(os.Stdin)
	write(response)
	if code != 0 {
		os.Exit(code)
	}
}

func processHostJSON(r io.Reader) (core.HostBridgeRPCResponse, int) {
	dec := json.NewDecoder(io.LimitReader(r, 64<<10))
	dec.DisallowUnknownFields()
	var req core.HostBridgeRPCRequest
	if err := dec.Decode(&req); err != nil {
		return core.HostBridgeRPCResponse{OK: false, Error: "invalid host bridge request: " + err.Error()}, 2
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return core.HostBridgeRPCResponse{OK: false, Error: "invalid host bridge request: multiple JSON values"}, 2
		}
		return core.HostBridgeRPCResponse{OK: false, Error: "invalid host bridge request: " + err.Error()}, 2
	}
	response, err := core.ApplyHostBridgeRPC(req)
	if err != nil {
		return core.HostBridgeRPCResponse{OK: false, Error: err.Error()}, 1
	}
	return response, 0
}
