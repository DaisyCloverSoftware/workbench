package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

// Private relay safe-hands actions let ordinary ChatGPT do the reasoning while
// Workbench supplies bounded repository eyes/hands. They deliberately exclude
// delegate_task and resolve_attention: autonomous work stays in relay/inbox and
// human answers stay in relay/answers so those boundaries remain explicit.
func isPrivateSafeHandsAction(action string) bool {
	switch action {
	case "list_projects", "list_files", "search_text", "read_file", "apply_patch", "run_safe_command", "save_note":
		return true
	default:
		return false
	}
}

// resolvePrivateSafeHandsProject adds a canonical-filesystem boundary on top of
// the relay's existing single-directory-name validation. Safe hands can read or
// modify source, so a symlink beneath WORKBENCH_RUNNER_ROOT must not be able to
// redirect a project outside that root (or alias a differently named project).
func resolvePrivateSafeHandsProject(name string) (string, error) {
	project, err := resolveProject(name)
	if err != nil {
		return "", err
	}
	root, err := filepath.EvalSymlinks(filepath.Dir(project))
	if err != nil {
		return "", errors.New("safe-hands runner root could not be canonicalised")
	}
	resolved, err := filepath.EvalSymlinks(project)
	if err != nil {
		return "", errors.New("safe-hands project could not be canonicalised")
	}
	root = filepath.Clean(root)
	resolved = filepath.Clean(resolved)
	if filepath.Dir(resolved) != root || filepath.Base(resolved) != filepath.Base(project) {
		return "", errors.New("safe-hands project resolves outside its authorised runner-root directory")
	}
	return resolved, nil
}

func executePrivateSafeHands(ctx context.Context, env privateControlEnvelope, mcpURL, authFile string) (map[string]any, error) {
	if env.Action == "list_projects" {
		if env.Project != "" {
			return nil, errors.New("list_projects does not accept a project")
		}
		if err := decodePrivateControlArgs(env.Args, &struct{}{}); err != nil {
			return nil, err
		}
		response, err := core.ApplyRunnerToolRequest(ctx, core.RunnerToolRequest{Action: "list_projects"})
		if err != nil {
			return nil, err
		}
		return map[string]any{"projects": response.Projects, "count": len(response.Projects)}, nil
	}

	project, err := resolvePrivateSafeHandsProject(env.Project)
	if err != nil {
		return nil, err
	}

	switch env.Action {
	case "list_files":
		var a struct {
			Subdir string `json:"subdir,omitempty"`
			Limit  int    `json:"limit,omitempty"`
		}
		if err := decodePrivateControlArgs(env.Args, &a); err != nil {
			return nil, err
		}
		if a.Limit <= 0 || a.Limit > 1000 {
			a.Limit = 500
		}
		return callMCP(ctx, mcpURL, authFile, "list_files", map[string]any{
			"project_path": project,
			"subdir":      strings.TrimSpace(a.Subdir),
			"limit":       a.Limit,
		})

	case "search_text":
		var a struct {
			Query  string `json:"query"`
			Subdir string `json:"subdir,omitempty"`
			Limit  int    `json:"limit,omitempty"`
		}
		if err := decodePrivateControlArgs(env.Args, &a); err != nil {
			return nil, err
		}
		a.Query = strings.TrimSpace(a.Query)
		if a.Query == "" {
			return nil, errors.New("search_text query is required")
		}
		if a.Limit <= 0 || a.Limit > 200 {
			a.Limit = 100
		}
		return callMCP(ctx, mcpURL, authFile, "search_text", map[string]any{
			"project_path": project,
			"query":        a.Query,
			"subdir":       strings.TrimSpace(a.Subdir),
			"limit":        a.Limit,
		})

	case "read_file":
		var a struct {
			Path      string `json:"path"`
			StartLine int    `json:"start_line,omitempty"`
			EndLine   int    `json:"end_line,omitempty"`
		}
		if err := decodePrivateControlArgs(env.Args, &a); err != nil {
			return nil, err
		}
		a.Path = strings.TrimSpace(a.Path)
		if a.Path == "" {
			return nil, errors.New("read_file path is required")
		}
		if a.StartLine < 0 || a.EndLine < 0 || (a.EndLine > 0 && a.StartLine > a.EndLine) {
			return nil, errors.New("read_file line range is invalid")
		}
		return callMCP(ctx, mcpURL, authFile, "read_file", map[string]any{
			"project_path": project,
			"path":         a.Path,
			"start_line":   a.StartLine,
			"end_line":     a.EndLine,
		})

	case "apply_patch":
		var a struct {
			Patch string `json:"patch"`
		}
		if err := decodePrivateControlArgs(env.Args, &a); err != nil {
			return nil, err
		}
		if strings.TrimSpace(a.Patch) == "" {
			return nil, errors.New("apply_patch patch is required")
		}
		return callMCP(ctx, mcpURL, authFile, "apply_patch", map[string]any{
			"project_path": project,
			"patch":        a.Patch,
		})

	case "run_safe_command":
		var a struct {
			Command string `json:"command"`
		}
		if err := decodePrivateControlArgs(env.Args, &a); err != nil {
			return nil, err
		}
		a.Command = strings.TrimSpace(a.Command)
		if a.Command == "" {
			return nil, errors.New("run_safe_command command is required")
		}
		return callMCP(ctx, mcpURL, authFile, "run_safe_command", map[string]any{
			"project_path": project,
			"command":      a.Command,
		})

	case "save_note":
		var a struct {
			Note string `json:"note"`
		}
		if err := decodePrivateControlArgs(env.Args, &a); err != nil {
			return nil, err
		}
		a.Note = strings.TrimSpace(a.Note)
		if a.Note == "" {
			return nil, errors.New("save_note note is required")
		}
		return callMCP(ctx, mcpURL, authFile, "save_note", map[string]any{
			"project_path": project,
			"note":         a.Note,
		})
	}

	return nil, errors.New("unsupported private safe-hands action")
}
