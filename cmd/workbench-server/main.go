package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
	"github.com/DaisyCloverSoftware/workbench/internal/mcp"
)

const serverVersion = "0.9.27"

func main() {
	port := flag.Int("port", 8765, "loopback MCP port")
	project := flag.String("project", "", "default project/repository path")
	tokenFile := flag.String("token-file", "", "0600 file containing the Authorization header value used by the local tunnel sidecar")
	flag.Parse()

	store, err := core.NewStore()
	if err != nil {
		fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		fatal(err)
	}
	if p := strings.TrimSpace(*project); p != "" {
		st := eng.State()
		if err := eng.SaveNotes(p, st.Notes); err != nil {
			fatal(err)
		}
	}

	st := eng.State()
	token := st.Preferences.MCPToken
	authDescription := "state-backed local bearer token"
	if path := strings.TrimSpace(*tokenFile); path != "" {
		token, err = loadOrCreateTunnelToken(path)
		if err != nil {
			fatal(err)
		}
		authDescription = "bearer token from " + path
	}

	srv := mcp.New(eng, *port, token)
	if err := srv.Start(); err != nil {
		fatal(err)
	}
	if err := eng.ResumeInterruptedTasks(); err != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = srv.Close(ctx)
		cancel()
		fatal(err)
	}

	fmt.Printf("Workbench MCP Server %s\n", serverVersion)
	fmt.Printf("MCP: %s\n", srv.URL())
	fmt.Printf("Workspace: %s\n", eng.State().ProjectPath)
	fmt.Printf("Auth: %s\n", authDescription)

	// On interactive runs Ctrl+C gets a graceful shutdown. Service managers may
	// terminate the process directly; the OS then closes the loopback listener.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Close(ctx)
}

// The file deliberately stores the complete HTTP Authorization value so
// tunnel-client can use `Authorization: file:/path` without placing the secret
// on argv. Workbench strips the Bearer prefix before comparing the token.
func loadOrCreateTunnelToken(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if b, readErr := os.ReadFile(path); readErr == nil {
		value := strings.TrimSpace(string(b))
		token := strings.TrimSpace(strings.TrimPrefix(value, "Bearer "))
		if token == "" {
			return "", fmt.Errorf("token file is empty: %s", path)
		}
		return token, nil
	} else if !os.IsNotExist(readErr) {
		return "", readErr
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	token := core.NewToken()
	if err := os.WriteFile(path, []byte("Bearer "+token+"\n"), 0o600); err != nil {
		return "", err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return "", err
	}
	return token, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "workbench-server:", err)
	os.Exit(1)
}
