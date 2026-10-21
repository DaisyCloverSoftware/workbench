package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func diagnosticFixture(t *testing.T, body any, status int) (string, string) {
	t.Helper()
	auth := filepath.Join(t.TempDir(), "auth")
	if err := os.WriteFile(auth, []byte("Bearer fixture-local-auth-value"), 0600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer fixture-local-auth-value" {
			t.Error("authentication contract changed")
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if request["method"] != "tools/call" {
			t.Errorf("unexpected method: %v", request["method"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if s, ok := body.(string); ok {
			_, _ = w.Write([]byte(s))
			return
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(server.Close)
	return server.URL, auth
}

func failedTool(message, output string) map[string]any {
	return map[string]any{"result": map[string]any{
		"isError": true, "structuredContent": map[string]any{"error": message, "output": output, "unrelated": "must not escape"},
	}}
}

func requireDiagnostic(t *testing.T, err error) mcpDiagnostic {
	t.Helper()
	var diagnosticErr *mcpDiagnosticError
	if !errors.As(err, &diagnosticErr) {
		t.Fatalf("missing typed diagnostic: %v", err)
	}
	return diagnosticErr.diagnostic
}

func TestMCPDiagnosticsPreserveToolFailuresWithoutInventingProcessFacts(t *testing.T) {
	cases := []struct{ name, message, output string }{
		{"policy", "command rejected by Workbench safe-command policy", ""},
		{"project", "project path is not a directory: /fixture/missing", ""},
		{"launch", "command failed: fork/exec /fixture/shell: no such file or directory", ""},
		{"nonzero", "command failed: exit status 127", "> sample build\n> tsc -b\nsh: tsc: not found"},
		{"command_timeout", "command failed: context deadline exceeded", "compiler started"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			url, auth := diagnosticFixture(t, failedTool(tc.message, tc.output), 200)
			result, err := callMCP(context.Background(), url, auth, "run_safe_command", map[string]any{"command": "npm test"})
			d := requireDiagnostic(t, err)
			if d.Layer != "mcp_tool" || d.Message != tc.message || d.Output != tc.output {
				t.Fatalf("lost diagnostic: %#v", d)
			}
			if d.ExecutionStarted != nil || d.ExitCode != nil || d.TimedOut != nil {
				t.Fatalf("invented producer metadata: %#v", d)
			}
			encoded, _ := json.Marshal(result)
			if strings.Contains(string(encoded), "must not escape") {
				t.Fatal("arbitrary structured data leaked")
			}
		})
	}
}

func TestMCPDiagnosticsSuccessfulOutputIsCompatible(t *testing.T) {
	url, auth := diagnosticFixture(t, map[string]any{"result": map[string]any{"structuredContent": map[string]any{"ok": true, "output": "4 tests passed"}}}, 200)
	result, err := callMCP(context.Background(), url, auth, "run_safe_command", map[string]any{"command": "npm test"})
	if err != nil || result["ok"] != true || result["output"] != "4 tests passed" || len(result) != 2 {
		t.Fatalf("success contract changed: %#v %v", result, err)
	}
}

func TestMCPDiagnosticsRPCValidationIsPreexecution(t *testing.T) {
	url, auth := diagnosticFixture(t, map[string]any{"error": map[string]any{"code": -32602, "message": "invalid params", "data": "unrelated private details"}}, 200)
	_, err := callMCP(context.Background(), url, auth, "run_safe_command", nil)
	d := requireDiagnostic(t, err)
	if d.Layer != "json_rpc" || d.ExecutionStarted == nil || *d.ExecutionStarted || d.RPCCode == nil || *d.RPCCode != -32602 {
		t.Fatalf("wrong RPC diagnostic: %#v", d)
	}
	if d.ExitCode != nil || strings.Contains(d.Message, "unrelated") {
		t.Fatal("unavailable facts or RPC data escaped")
	}
}

func TestMCPDiagnosticsRequestAndAuthValidation(t *testing.T) {
	_, err := callMCP(context.Background(), "http://unused", filepath.Join(t.TempDir(), "missing"), "run_safe_command", nil)
	d := requireDiagnostic(t, err)
	if d.ExecutionStarted == nil || *d.ExecutionStarted || d.Layer != "mcp_preflight" {
		t.Fatalf("wrong authentication failure: %#v", d)
	}
	url, auth := diagnosticFixture(t, nil, 200)
	_, err = callMCP(context.Background(), url, auth, "run_safe_command", map[string]any{"invalid": func() {}})
	d = requireDiagnostic(t, err)
	if d.ExecutionStarted == nil || *d.ExecutionStarted || d.Layer != "request_validation" {
		t.Fatalf("wrong encoding failure: %#v", d)
	}
}

func TestMCPDiagnosticsTransportAndUnusableResponsesStayUnknown(t *testing.T) {
	cases := []struct {
		name, body, layer string
		status            int
	}{
		{"http", "do not echo HTTP bodies", "http_response", 503},
		{"malformed", "not-json", "response_decode", 200},
		{"missing_result", `{}`, "response_shape", 200},
		{"too_large", strings.Repeat("x", maxMCPResponseBytes+1), "response_limit", 200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			url, auth := diagnosticFixture(t, tc.body, tc.status)
			_, err := callMCP(context.Background(), url, auth, "run_safe_command", nil)
			d := requireDiagnostic(t, err)
			if d.Layer != tc.layer || d.ExecutionStarted != nil || d.ExitCode != nil || d.Output != "" {
				t.Fatalf("false execution claim: %#v", d)
			}
			if (tc.name == "too_large") != d.ResponseTruncated {
				t.Fatalf("missing response limit flag: %#v", d)
			}
		})
	}
	url, auth := diagnosticFixture(t, nil, 200)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	_, err := callMCP(ctx, url, auth, "run_safe_command", nil)
	d := requireDiagnostic(t, err)
	if d.Layer != "transport_timeout" || d.TimedOut == nil || !*d.TimedOut || d.ExecutionStarted != nil || d.ExitCode != nil {
		t.Fatalf("wrong timeout: %#v", d)
	}
	canceled, stop := context.WithCancel(context.Background())
	stop()
	_, err = callMCP(canceled, url, auth, "run_safe_command", nil)
	if d = requireDiagnostic(t, err); d.Layer != "transport_canceled" || d.ExecutionStarted != nil {
		t.Fatalf("wrong cancellation: %#v", d)
	}
}

func TestMCPDiagnosticsRedactionBeforeTruncation(t *testing.T) {
	t.Setenv("WORKBENCH_FIXTURE_PASSWORD", "fixture-sensitive-environment-value")
	token := "ghp_" + strings.Repeat("x", 32)
	text := "useful compiler error\nAuthorization: Bearer fixture-local-auth-value\n" + token + "\nDB_PASSWORD=tiny\n" +
		"environment echoed fixture-sensitive-environment-value\nhttps://example:password@example.invalid/private\n" +
		"-----BEGIN PRIVATE KEY-----\nkey-body-sentinel\n-----END PRIVATE KEY-----\n" + strings.Repeat("界", maxMCPDiagnosticOutput)
	url, auth := diagnosticFixture(t, failedTool("build failed", text), 200)
	_, err := callMCP(context.Background(), url, auth, "run_safe_command", nil)
	d := requireDiagnostic(t, err)
	if !d.Redacted || !d.OutputTruncated || len(d.Output) > maxMCPDiagnosticOutput || !utf8.ValidString(d.Output) {
		t.Fatalf("unbounded/invalid diagnostic: %#v", d)
	}
	if !strings.Contains(d.Output, "useful compiler error") {
		t.Fatal("useful non-sensitive output discarded")
	}
	for _, secret := range []string{token, "fixture-local-auth-value", "tiny", "fixture-sensitive-environment-value", "key-body-sentinel", "example:password@"} {
		if strings.Contains(d.Output, secret) {
			t.Errorf("sensitive fixture escaped: %q", secret)
		}
	}
	d = newMCPDiagnostic("mcp_tool", strings.Repeat("m", maxMCPDiagnosticMessage+1), "")
	if !d.MessageTruncated || len(d.Message) > maxMCPDiagnosticMessage {
		t.Fatal("message not bounded")
	}
}

func TestMCPDiagnosticsPrivateOutboxKeepsFailureAndCorrelation(t *testing.T) {
	url, auth := diagnosticFixture(t, failedTool("command failed: exit status 2", "TS2307: Cannot find module"), 200)
	_, err := callMCP(context.Background(), url, auth, "run_safe_command", nil)
	out := privateControlOutbox{Version: 1, ID: "fixture-request-123", Action: "run_safe_command", Status: "failed"}
	retainMCPDiagnostic(&out, err)
	raw, marshalErr := marshalPrivateControlOutbox(out)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	var parsed struct {
		Status string `json:"status"`
		Result struct {
			Diagnostic mcpDiagnostic `json:"diagnostic"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Status != "failed" || parsed.Result.Diagnostic.RequestID != out.ID || !strings.Contains(parsed.Result.Diagnostic.Output, "TS2307") {
		t.Fatalf("diagnostic lost at outbox: %s", raw)
	}
}

func TestMCPDiagnosticsCandidatePreflightDoesNotInvokeMCP(t *testing.T) {
	candidate := privateControlCandidate{id: "fixture-request-456", env: privateControlEnvelope{
		Version: 1, ID: "fixture-request-456", Action: "run_safe_command", Project: "../outside",
		Args: json.RawMessage(`{"command":"npm test"}`),
	}}
	out := executePrivateControlCandidate(context.Background(), candidate, "", "http://must-not-be-called.invalid", "missing")
	raw, err := marshalPrivateControlOutbox(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"layer": "relay_preflight"`) || !strings.Contains(string(raw), `"execution_started": false`) || out.Status != "failed" {
		t.Fatalf("missing preflight result: %s", raw)
	}
}

func TestMCPDiagnosticsDoNotBroadenSafeCommandPolicy(t *testing.T) {
	for _, cmd := range []string{"npm test --prefix apps/sample", "npm run build --prefix apps/sample"} {
		if !core.IsSafeCommand(cmd) {
			t.Errorf("documented command unexpectedly rejected: %s", cmd)
		}
	}
	for _, cmd := range []string{"npm run deploy", "npm test; echo unexpected", "git push", "sh -c npm-test"} {
		if core.IsSafeCommand(cmd) {
			t.Errorf("policy widened: %s", cmd)
		}
	}
}
