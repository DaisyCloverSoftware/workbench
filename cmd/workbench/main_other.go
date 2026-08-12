//go:build !windows

package main

import (
	"fmt"
	"os"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
	"github.com/DaisyCloverSoftware/workbench/internal/mcp"
)

func main() {
	store, err := core.NewStore()
	if err != nil {
		panic(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		panic(err)
	}
	st := eng.State()
	s := mcp.New(eng, st.Preferences.MCPPort, st.Preferences.MCPToken)
	if err := s.Start(); err != nil {
		panic(err)
	}
	fmt.Println("Workbench headless MCP", s.URL())
	fmt.Println("Desktop UI is available in the Windows build.")
	select {}
	_ = os.Stdout
}
