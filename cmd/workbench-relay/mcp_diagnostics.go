package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

const maxMCPDiagnosticOutput = 16 << 10
const maxMCPDiagnosticMessage = 4 << 10
const maxMCPResponseBytes = 1 << 20

// Null execution fields mean the producer did not report that fact. In
// particular, an HTTP/MCP response is not evidence that a process started.
// The current safe-command MCP supplies error/output, not process metadata.
type mcpDiagnostic struct {
	Version           int    `json:"version"`
	RequestID         string `json:"request_id,omitempty"`
	Layer             string `json:"layer"`
	ExecutionStarted  *bool  `json:"execution_started"`
	ExitCode          *int   `json:"exit_code"`
	TimedOut          *bool  `json:"timed_out"`
	Message           string `json:"message"`
	Output            string `json:"output,omitempty"`
	OutputTruncated   bool   `json:"output_truncated"`
	MessageTruncated  bool   `json:"message_truncated"`
	ResponseTruncated bool   `json:"response_truncated"`
	Redacted          bool   `json:"redacted"`
	RPCCode           *int   `json:"rpc_code,omitempty"`
}

type mcpDiagnosticError struct{ diagnostic mcpDiagnostic }

func (e *mcpDiagnosticError) Error() string {
	return "local MCP failure (" + e.diagnostic.Layer + "): " + e.diagnostic.Message
}

func diagnosticFailure(d mcpDiagnostic) (map[string]any, error) {
	e := &mcpDiagnosticError{diagnostic: d}
	return map[string]any{"ok": false, "diagnostic": d}, e
}

func newMCPDiagnostic(layer, message, output string, sensitive ...string) mcpDiagnostic {
	m, mt, mr := sanitizeMCPDiagnostic(message, maxMCPDiagnosticMessage, sensitive...)
	o, ot, or := sanitizeMCPDiagnostic(output, maxMCPDiagnosticOutput, sensitive...)
	return mcpDiagnostic{Version: 1, Layer: layer, Message: m, Output: o,
		MessageTruncated: mt, OutputTruncated: ot, Redacted: mr || or}
}

func preflightDiagnostic(layer, message string) mcpDiagnostic {
	d := newMCPDiagnostic(layer, message, "")
	started := false
	d.ExecutionStarted = &started
	return d
}

// Keep the existing 20-second transport deadline and authentication contract.
// A transport failure leaves process execution unknown; it must not trigger a
// replay. No response body, RPC data, request args or environment is dumped.
func callMCPDiagnosed(ctx context.Context, url, authFile, tool string, args map[string]any) (map[string]any, error) {
	auth, err := os.ReadFile(authFile)
	if err != nil {
		cause := err
		var pathErr *os.PathError
		if errors.As(err, &pathErr) {
			cause = pathErr.Err
		}
		return diagnosticFailure(preflightDiagnostic("mcp_preflight", "read local MCP authentication file: "+cause.Error()))
	}
	sensitive := []string{strings.TrimSpace(string(auth))}
	if fields := strings.Fields(string(auth)); len(fields) > 1 {
		sensitive = append(sensitive, fields[len(fields)-1])
	}
	payload, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{"name": tool, "arguments": args},
	})
	if err != nil {
		return diagnosticFailure(preflightDiagnostic("request_validation", "encode MCP arguments: "+err.Error()))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return diagnosticFailure(preflightDiagnostic("mcp_preflight", "invalid MCP request URL"))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", strings.TrimSpace(string(auth)))
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		d := newMCPDiagnostic("transport", "MCP transport failed; execution status is unknown", "")
		var netErr net.Error
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
			yes := true
			d.TimedOut = &yes
			d.Layer = "transport_timeout"
			d.Message = "MCP transport timed out; execution status is unknown; do not replay automatically"
		} else if errors.Is(err, context.Canceled) {
			d.Layer = "transport_canceled"
			d.Message = "MCP transport was canceled; execution status is unknown"
		}
		return diagnosticFailure(d)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return diagnosticFailure(newMCPDiagnostic("http_response", fmt.Sprintf("local MCP returned HTTP %d; execution status is unknown", resp.StatusCode), ""))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxMCPResponseBytes+1))
	if err != nil {
		return diagnosticFailure(newMCPDiagnostic("response_read", "MCP response could not be read completely; execution status is unknown", ""))
	}
	if len(body) > maxMCPResponseBytes {
		d := newMCPDiagnostic("response_limit", "MCP response exceeded the bounded read limit; execution status is unknown", "")
		d.ResponseTruncated = true
		return diagnosticFailure(d)
	}
	var rr rpcResponse
	if err := json.Unmarshal(body, &rr); err != nil {
		return diagnosticFailure(newMCPDiagnostic("response_decode", "MCP response was not valid JSON; execution status is unknown", ""))
	}
	if rr.Error != nil {
		message := "MCP returned a JSON-RPC error without a message"
		var code *int
		if v, ok := rr.Error.(map[string]any); ok {
			if s, ok := v["message"].(string); ok {
				message = s
			}
			if n, ok := v["code"].(float64); ok && n >= -32768 && n <= 32767 && n == float64(int(n)) {
				c := int(n)
				code = &c
			}
		}
		d := newMCPDiagnostic("json_rpc", message, "", sensitive...)
		d.RPCCode = code
		if code != nil && (*code == -32700 || *code == -32600 || *code == -32601 || *code == -32602) {
			no := false
			d.ExecutionStarted = &no
		}
		return diagnosticFailure(d)
	}
	if rr.Result.IsError {
		// Deliberately allow only the two documented error fields. In particular,
		// do not forward arbitrary structuredContent or RPC error.data.
		message, _ := rr.Result.StructuredContent["error"].(string)
		output, _ := rr.Result.StructuredContent["output"].(string)
		if strings.TrimSpace(message) == "" {
			message = "MCP tool reported an error without structured error text"
		}
		return diagnosticFailure(newMCPDiagnostic("mcp_tool", message, output, sensitive...))
	}
	if rr.Result.StructuredContent == nil {
		return diagnosticFailure(newMCPDiagnostic("response_shape", "MCP returned no structured result; execution status is unknown", ""))
	}
	// Successful consumers retain their existing shape and secret/size guards.
	return rr.Result.StructuredContent, nil
}

// The private control envelope already correlates success/failure with its ID.
// Add the same identifier to preserved failure diagnostics, without changing
// the legacy failed status or concise top-level error field.
func retainMCPDiagnostic(out *privateControlOutbox, err error) {
	var diagnosticErr *mcpDiagnosticError
	if errors.As(err, &diagnosticErr) {
		d := diagnosticErr.diagnostic
		d.RequestID = out.ID
		out.Result = map[string]any{"ok": false, "diagnostic": d}
		out.Error = diagnosticErr.Error()
	} else if out.Action == "run_safe_command" {
		// This action has only project/argument validation before callMCP. Once
		// invoked, callMCPDiagnosed returns a typed diagnostic for every error.
		d := preflightDiagnostic("relay_preflight", err.Error())
		d.RequestID = out.ID
		out.Result = map[string]any{"ok": false, "diagnostic": d}
		out.Error = "safe-command preflight failed: " + d.Message
	}
}

var diagnosticSensitiveKey = regexp.MustCompile(`(?i)(?:[a-z0-9_-]*(?:password|passwd|token|secret|credential|api[_-]?key|private[_-]?key)[a-z0-9_-]*["']?\s*[:=]|\b(?:authorization|proxy-authorization|cookie|set-cookie)\s*:)`)
var diagnosticEnvironmentAssignment = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*=`)
var diagnosticURLCredential = regexp.MustCompile(`(?i)https?://[^\s/]+@`)
var diagnosticANSI = regexp.MustCompile("\x1b\\[[0-?]*[ -/]*[@-~]")

func sanitizeMCPDiagnostic(text string, limit int, sensitive ...string) (string, bool, bool) {
	// Use sensitive environment values only for masking, never for serialization.
	for _, entry := range os.Environ() {
		if key, value, ok := strings.Cut(entry, "="); ok && value != "" && diagnosticSensitiveKey.MatchString(key+"=") {
			sensitive = append(sensitive, value)
		}
	}
	text = diagnosticANSI.ReplaceAllString(strings.ToValidUTF8(text, "�"), "")
	text = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\t' && r != '\r' {
			return -1
		}
		return r
	}, text)
	lines := strings.Split(strings.ReplaceAll(text, "\r", "\n"), "\n")
	redacted, inKey := false, false
	for i, line := range lines {
		upper := strings.ToUpper(line)
		if strings.Contains(upper, "-----BEGIN") && strings.Contains(upper, "PRIVATE KEY-----") {
			inKey = true
		}
		hide := inKey || core.LooksSecret(line) || diagnosticSensitiveKey.MatchString(line) || diagnosticEnvironmentAssignment.MatchString(strings.TrimSpace(line)) || diagnosticURLCredential.MatchString(line)
		for _, value := range sensitive {
			if value != "" && strings.Contains(line, value) {
				hide = true
			}
		}
		if hide {
			lines[i] = "[redacted sensitive diagnostic line]"
			redacted = true
		}
		if inKey && strings.Contains(upper, "-----END") && strings.Contains(upper, "PRIVATE KEY-----") {
			inKey = false
		}
	}
	safe := strings.TrimSpace(strings.Join(lines, "\n"))
	truncated := len(safe) > limit
	if truncated {
		safe = safe[:limit]
		for !utf8.ValidString(safe) {
			safe = safe[:len(safe)-1]
		}
	}
	return safe, truncated, redacted
}
