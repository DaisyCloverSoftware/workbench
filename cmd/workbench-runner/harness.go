package main

import (
	"errors"
	"os"
	"strings"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

type runnerHarnessCommandResponse struct {
	OK          bool   `json:"ok"`
	Action      string `json:"action,omitempty"`
	Configured  bool   `json:"configured,omitempty"`
	Available   bool   `json:"available,omitempty"`
	AdapterName string `json:"adapter_name,omitempty"`
	Status      string `json:"status,omitempty"`
	Deleted     bool   `json:"deleted,omitempty"`
	Error       string `json:"error,omitempty"`
}

var (
	saveRunnerHarnessAdapter   = core.SaveRunnerHarnessAdapter
	deleteRunnerHarnessAdapter = core.DeleteRunnerHarnessAdapter
	runnerHarnessStatus        = core.RunnerHarnessConfigurationStatus
)

func harness() {
	response, err := applyHarnessCommand(os.Args[2:])
	if err != nil {
		response.OK = false
		response.Error = err.Error()
		write(response)
		os.Exit(1)
	}
	write(response)
}

func applyHarnessCommand(args []string) (runnerHarnessCommandResponse, error) {
	if len(args) == 0 {
		return runnerHarnessCommandResponse{}, errors.New("runner harness requires get, set or delete")
	}
	action := strings.ToLower(strings.TrimSpace(args[0]))
	switch action {
	case "get":
		if len(args) != 1 {
			return runnerHarnessCommandResponse{}, errors.New("usage: workbench-runner harness get")
		}
		return harnessStatusResponse(action, false), nil
	case "set":
		if len(args) != 2 || strings.TrimSpace(args[1]) == "" {
			return runnerHarnessCommandResponse{}, errors.New("usage: workbench-runner harness set <adapter-executable>")
		}
		if _, err := saveRunnerHarnessAdapter(args[1]); err != nil {
			return runnerHarnessCommandResponse{}, err
		}
		return harnessStatusResponse(action, false), nil
	case "delete":
		if len(args) != 1 {
			return runnerHarnessCommandResponse{}, errors.New("usage: workbench-runner harness delete")
		}
		if err := deleteRunnerHarnessAdapter(); err != nil {
			return runnerHarnessCommandResponse{}, err
		}
		return harnessStatusResponse(action, true), nil
	default:
		return runnerHarnessCommandResponse{}, errors.New("runner harness action must be get, set or delete")
	}
}

func harnessStatusResponse(action string, deleted bool) runnerHarnessCommandResponse {
	status := runnerHarnessStatus()
	return runnerHarnessCommandResponse{
		OK:          true,
		Action:      action,
		Configured:  status.Configured,
		Available:   status.Available,
		AdapterName: status.AdapterName,
		Status:      status.Status,
		Deleted:     deleted,
	}
}
