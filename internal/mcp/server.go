package mcp

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

type Server struct {
	engine *core.Engine
	token  string
	port   int
	http   *http.Server
}

func New(engine *core.Engine, port int, token string) *Server {
	if port == 0 {
		port = 8765
	}
	return &Server{engine: engine, port: port, token: token}
}

func (s *Server) URL() string { return "http://127.0.0.1:" + strconv.Itoa(s.port) + "/mcp" }

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.handle)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.RemoteAddr != "" {
			host, _, _ := net.SplitHostPort(r.RemoteAddr)
			if host != "127.0.0.1" && host != "::1" {
				http.Error(w, "local only", http.StatusForbidden)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"name":"Workbench MCP"}`))
	})
	addr := "127.0.0.1:" + strconv.Itoa(s.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.http = &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = s.http.Serve(ln) }()
	return nil
}

func (s *Server) Close(ctx context.Context) error {
	if s.http == nil {
		return nil
	}
	return s.http.Shutdown(ctx)
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" && origin != "http://127.0.0.1" && origin != "http://localhost" {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	if s.token != "" {
		got := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if subtle.ConstantTimeCompare([]byte(got), []byte(s.token)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 16<<20))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var req rpcRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeRPC(w, rpcResponse{JSONRPC: "2.0", ID: nil, Error: &rpcError{Code: -32700, Message: "parse error"}})
		return
	}
	if req.JSONRPC != "2.0" {
		s.writeRPC(w, rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32600, Message: "invalid request"}})
		return
	}
	if strings.HasPrefix(req.Method, "notifications/") {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	var result any
	switch req.Method {
	case "initialize":
		result = map[string]any{
			"protocolVersion": "2025-11-25",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "Workbench", "version": "0.4.0-dev"},
			"instructions": "Workbench gives ordinary Chat safe repository eyes and hands plus autonomous workers. Start with get_workspace. Use list_files, search_text, and read_file to understand code without spending an autonomous-agent allowance. Prefer apply_patch and run_safe_command when Chat can supply the intelligence. Use delegate_task only when autonomous exploration is genuinely useful. After delegation, poll get_task yourself until completed, failed, or needs_attention. Never ask the human to watch progress. Surface needs_attention only when Workbench reports a genuine human decision boundary.",
		}
	case "ping":
		result = map[string]any{}
	case "tools/list":
		result = map[string]any{"tools": toolsList()}
	case "tools/call":
		var p struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			s.writeRPC(w, rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32602, Message: "invalid params"}})
			return
		}
		result = s.callTool(r.Context(), p.Name, p.Arguments)
	default:
		s.writeRPC(w, rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32601, Message: "method not found"}})
		return
	}
	s.writeRPC(w, rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result})
}

func (s *Server) writeRPC(w http.ResponseWriter, v rpcResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("MCP-Protocol-Version", "2025-11-25")
	_ = json.NewEncoder(w).Encode(v)
}

func annotations(readOnly, destructive, openWorld bool) map[string]any {
	return map[string]any{
		"readOnlyHint":    readOnly,
		"destructiveHint": destructive,
		"openWorldHint":   openWorld,
	}
}

func tool(name, title, description string, input, output map[string]any, ann map[string]any) map[string]any {
	return map[string]any{
		"name":         name,
		"title":        title,
		"description":  description,
		"inputSchema":  input,
		"outputSchema": output,
		"annotations":  ann,
	}
}

func toolsList() []map[string]any {
	project := strProp("Absolute project/repository path. Omit to use the active Workbench workspace.")
	subdir := strProp("Optional relative subdirectory within the active project.")
	return []map[string]any{
		tool("get_workspace", "Get Workbench workspace", "Inspect the active project, routing policy, and available workers before deciding how to execute a development request.", objSchema(map[string]any{}, nil), anyObjectSchema(), annotations(true, false, false)),
		tool("list_files", "List repository files", "List model-safe source files under the active project without traversing generated trees or credential directories. Use this before delegating merely to discover the repository layout.", objSchema(map[string]any{"project_path": project, "subdir": subdir, "limit": intProp("Maximum files to return (1-1000).")}, nil), anyObjectSchema(), annotations(true, false, false)),
		tool("search_text", "Search repository text", "Search model-safe source files for text. Binary, oversized, generated, credential-like, and probable-secret files are skipped.", objSchema(map[string]any{"project_path": project, "query": strProp("Text to find, case-insensitive"), "subdir": subdir, "limit": intProp("Maximum matching lines to return (1-200).")}, []string{"query"}), anyObjectSchema(), annotations(true, false, false)),
		tool("read_file", "Read repository file", "Read a line range from one model-safe source file. Paths must stay inside the active project; credential files, binary files, oversized files, and files containing probable secret material are refused.", objSchema(map[string]any{"project_path": project, "path": strProp("Relative file path inside the project"), "start_line": intProp("Optional 1-based first line"), "end_line": intProp("Optional 1-based last line")}, []string{"path"}), anyObjectSchema(), annotations(true, false, false)),
		tool("delegate_task", "Delegate autonomous coding task", "Delegate a genuinely autonomous coding task. Workbench routes zero-marginal and included-subscription workers before scarce Work/Codex and leaves metered APIs disabled unless explicitly enabled.", objSchema(map[string]any{"intent": strProp("Outcome to achieve"), "project_path": project}, []string{"intent"}), anyObjectSchema(), annotations(false, false, false)),
		tool("get_task", "Get task status", "Read durable task status. Poll this yourself after delegation; only surface a needs_attention question to the human.", objSchema(map[string]any{"task_id": strProp("Workbench task id")}, []string{"task_id"}), anyObjectSchema(), annotations(true, false, false)),
		tool("list_tasks", "List Workbench tasks", "List recent Workbench tasks and their statuses.", objSchema(map[string]any{}, nil), anyObjectSchema(), annotations(true, false, false)),
		tool("apply_patch", "Apply Chat-generated patch", "Apply a unified git patch supplied by ordinary Chat. Prefer this no-Work path when Chat can reason out the code and Workbench only needs to provide hands.", objSchema(map[string]any{"project_path": project, "patch": strProp("Unified git patch")}, []string{"patch"}), anyObjectSchema(), annotations(false, false, false)),
		tool("run_safe_command", "Run safe development command", "Run a non-destructive local build, test, lint, status, or diff command under Workbench's allowlist. No deploy, push, network shell, or destructive commands.", objSchema(map[string]any{"project_path": project, "command": strProp("Safe command, e.g. git diff, npm test, go test ./...")}, []string{"command"}), anyObjectSchema(), annotations(false, false, false)),
		tool("save_note", "Save project note", "Save a non-secret project note in Workbench. Secret-like text is refused and must be stored through the local encrypted vault instead.", objSchema(map[string]any{"project_path": project, "note": strProp("Note text")}, []string{"note"}), anyObjectSchema(), annotations(false, false, false)),
		tool("resolve_attention", "Resolve human attention request", "Resume a paused task after the human has answered a genuine Workbench decision or permission request.", objSchema(map[string]any{"task_id": strProp("Task id"), "answer": strProp("Human decision")}, []string{"task_id", "answer"}), anyObjectSchema(), annotations(false, false, false)),
	}
}

func strProp(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func intProp(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

func objSchema(props map[string]any, req []string) map[string]any {
	m := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
	if len(req) > 0 {
		m["required"] = req
	}
	return m
}

func anyObjectSchema() map[string]any { return map[string]any{"type": "object", "additionalProperties": true} }

func textContent(v any, isErr bool) map[string]any {
	b, _ := json.MarshalIndent(v, "", "  ")
	return map[string]any{
		"structuredContent": v,
		"content":           []map[string]any{{"type": "text", "text": string(b)}},
		"isError":           isErr,
	}
}

func arg(a map[string]any, k string) string {
	if v, ok := a[k].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func intArg(a map[string]any, k string) int {
	switch v := a[k].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case json.Number:
		n, _ := strconv.Atoi(v.String())
		return n
	default:
		return 0
	}
}

func (s *Server) projectArg(a map[string]any) string {
	if p := arg(a, "project_path"); p != "" {
		return p
	}
	return strings.TrimSpace(s.engine.State().ProjectPath)
}

func (s *Server) requireProject(a map[string]any) (string, any) {
	project := s.projectArg(a)
	if project == "" {
		return "", textContent(map[string]any{"error": "no active Workbench project; provide project_path or configure the server default workspace"}, true)
	}
	return project, nil
}

func (s *Server) callTool(ctx context.Context, name string, a map[string]any) any {
	switch name {
	case "get_workspace":
		st := s.engine.State()
		providers := s.engine.Providers()
		workers := make([]map[string]any, 0, len(providers))
		for _, p := range providers {
			workers = append(workers, map[string]any{
				"id":            p.ID,
				"name":          p.Name,
				"capability":    p.Capability,
				"installed":     p.Installed,
				"authenticated": p.Authenticated,
				"status":        p.Status,
				"cost":          p.Cost,
				"priority":      p.Priority,
			})
		}
		return textContent(map[string]any{
			"project_path":      st.ProjectPath,
			"avoid_work_usage":  st.Preferences.AvoidWorkUsage,
			"allow_metered_api": st.Preferences.AllowMeteredAPI,
			"autonomy_mode":     st.Preferences.AutonomyMode,
			"connected_workers": workers,
		}, false)

	case "list_files":
		project, errResult := s.requireProject(a)
		if errResult != nil {
			return errResult
		}
		files, err := core.ListProjectFiles(project, arg(a, "subdir"), intArg(a, "limit"))
		if err != nil {
			return textContent(map[string]any{"error": err.Error()}, true)
		}
		return textContent(map[string]any{"project_path": project, "files": files, "count": len(files)}, false)

	case "search_text":
		project, errResult := s.requireProject(a)
		if errResult != nil {
			return errResult
		}
		hits, err := core.SearchProjectText(project, arg(a, "query"), arg(a, "subdir"), intArg(a, "limit"))
		if err != nil {
			return textContent(map[string]any{"error": err.Error()}, true)
		}
		return textContent(map[string]any{"project_path": project, "query": arg(a, "query"), "hits": hits, "count": len(hits)}, false)

	case "read_file":
		project, errResult := s.requireProject(a)
		if errResult != nil {
			return errResult
		}
		content, err := core.ReadProjectFile(project, arg(a, "path"), intArg(a, "start_line"), intArg(a, "end_line"))
		if err != nil {
			return textContent(map[string]any{"error": err.Error()}, true)
		}
		return textContent(map[string]any{"project_path": project, "path": arg(a, "path"), "content": content}, false)

	case "delegate_task":
		project, errResult := s.requireProject(a)
		if errResult != nil {
			return errResult
		}
		t, err := s.engine.Delegate("chatgpt-mcp", arg(a, "intent"), project)
		if err != nil {
			return textContent(map[string]any{"error": err.Error()}, true)
		}
		return textContent(map[string]any{
			"task_id":      t.ID,
			"status":       t.Status,
			"project_path": project,
			"instruction":  "Poll get_task until completed, failed, or needs_attention. Do not ask the human to check progress.",
		}, false)

	case "get_task":
		t, ok := s.engine.Task(arg(a, "task_id"))
		if !ok {
			return textContent(map[string]any{"error": "task not found"}, true)
		}
		return textContent(t, false)

	case "list_tasks":
		st := s.engine.State()
		tasks := st.Tasks
		if len(tasks) > 20 {
			tasks = tasks[:20]
		}
		return textContent(map[string]any{"tasks": tasks}, false)

	case "apply_patch":
		project, errResult := s.requireProject(a)
		if errResult != nil {
			return errResult
		}
		out, err := core.ApplyPatch(ctx, project, arg(a, "patch"))
		if err != nil {
			return textContent(map[string]any{"error": err.Error(), "output": out}, true)
		}
		return textContent(map[string]any{"ok": true, "project_path": project, "output": out}, false)

	case "run_safe_command":
		project, errResult := s.requireProject(a)
		if errResult != nil {
			return errResult
		}
		out, err := core.RunSafeCommand(ctx, project, arg(a, "command"))
		if err != nil {
			return textContent(map[string]any{"error": err.Error(), "output": out}, true)
		}
		return textContent(map[string]any{"ok": true, "project_path": project, "output": out}, false)

	case "save_note":
		note := arg(a, "note")
		if core.LooksSecret(note) {
			return textContent(map[string]any{"error": "note appears to contain a secret; Workbench refused to expose/store it through MCP. Use the local encrypted vault."}, true)
		}
		project := s.projectArg(a)
		st := s.engine.State()
		merged := strings.TrimSpace(st.Notes)
		if merged != "" {
			merged += "\n\n"
		}
		merged += "[" + time.Now().Format("2006-01-02 15:04") + "] " + note
		if err := s.engine.SaveNotes(project, merged); err != nil {
			return textContent(map[string]any{"error": err.Error()}, true)
		}
		return textContent(map[string]any{"ok": true, "project_path": project}, false)

	case "resolve_attention":
		if err := s.engine.ResolveAttention(arg(a, "task_id"), arg(a, "answer")); err != nil {
			return textContent(map[string]any{"error": err.Error()}, true)
		}
		return textContent(map[string]any{"ok": true, "instruction": "Poll get_task again until terminal state."}, false)

	default:
		return textContent(map[string]any{"error": fmt.Sprintf("unknown tool %q", name)}, true)
	}
}
