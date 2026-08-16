package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func toolJSON() {
	dec := json.NewDecoder(os.Stdin)
	dec.DisallowUnknownFields()
	var req core.RunnerToolRequest
	if err := dec.Decode(&req); err != nil {
		write(core.RunnerToolResponse{Error: "invalid runner tool request"})
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	response, err := core.ApplyRunnerToolRequest(ctx, req)
	if err != nil {
		if strings.TrimSpace(response.Error) == "" {
			response.Error = "runner tool operation failed"
		}
		write(response)
		os.Exit(1)
	}
	write(response)
}
