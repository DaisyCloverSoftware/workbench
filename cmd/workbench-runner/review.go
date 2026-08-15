package main

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func reviewJSON() {
	dec := json.NewDecoder(os.Stdin)
	dec.DisallowUnknownFields()
	var req core.RunnerReviewRequest
	if err := dec.Decode(&req); err != nil {
		write(core.RunnerReviewResponse{OK: false, Error: "invalid runner review request"})
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	response, err := core.ApplyRunnerReviewRequest(ctx, req)
	if err != nil {
		write(core.RunnerReviewResponse{OK: false, Error: err.Error()})
		os.Exit(1)
	}
	write(response)
}
