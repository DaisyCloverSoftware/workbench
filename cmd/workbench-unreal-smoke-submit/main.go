package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: workbench-unreal-smoke-submit <windows-host-id>")
		os.Exit(2)
	}
	job, err := core.SubmitUnrealSmokeJob(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "workbench-unreal-smoke-submit:", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(job); err != nil {
		fmt.Fprintln(os.Stderr, "workbench-unreal-smoke-submit:", err)
		os.Exit(1)
	}
}
