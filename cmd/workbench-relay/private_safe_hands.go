package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

// Private relay safe controls let ordinary ChatGPT do the reasoning while
// Workbench supplies bounded repository eyes/hands, direct structured machine
// commands, committed operations scripts, and privacy-safe maintenance status.
// Read-only task diagnostics are included so a lead chat can inspect a
// long-running supervised operation without asking the human to watch it.
// They deliberately exclude delegate_task, cancellation and resolve_attention:
// autonomous AI work stays in relay/inbox and human answers stay in
// relay/answers. Routine machine work does not need an AI worker at all.
func isPrivateSafeHandsAction(action string) bool {
	switch action {
	case "update_status", "get_task", "list_tasks", "list_projects", "ensure_github_project", "list_files", "search_text", "read_file", "apply_patch", "run_safe_command", "inspect_machine", "inspect_machine_batch", "run_machine_command", "run_operations_script", "save_note", "list_windows_hosts", "run_windows_blender_version", "get_windows_host_job":
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
	if env.Action == "update_status" {
		if env.Project != "" {
			return nil, errors.New("update_status does not accept a project")
		}
		if err := decodePrivateControlArgs(env.Args, &struct{}{}); err != nil {
			return nil, err
		}
		return readPrivateUpdateStatus()
	}

	if env.Action == "get_task" {
		if env.Project != "" {
			return nil, errors.New("get_task does not accept a project")
		}
		var a struct {
			TaskID string `json:"task_id"`
		}
		if err := decodePrivateControlArgs(env.Args, &a); err != nil {
			return nil, err
		}
		a.TaskID = strings.TrimSpace(a.TaskID)
		if a.TaskID == "" || len(a.TaskID) > 160 {
			return nil, errors.New("get_task task_id is required and must be bounded")
		}
		return callMCP(ctx, mcpURL, authFile, "get_task", map[string]any{"task_id": a.TaskID})
	}

	if env.Action == "list_tasks" {
		if env.Project != "" {
			return nil, errors.New("list_tasks does not accept a project")
		}
		if err := decodePrivateControlArgs(env.Args, &struct{}{}); err != nil {
			return nil, err
		}
		return callMCP(ctx, mcpURL, authFile, "list_tasks", map[string]any{})
	}

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

	if env.Action == "list_windows_hosts" {
		if env.Project != "" {
			return nil, errors.New("list_windows_hosts does not accept a project")
		}
		if err := decodePrivateControlArgs(env.Args, &struct{}{}); err != nil {
			return nil, err
		}
		hosts, err := core.ListHostBridgeHosts()
		if err != nil {
			return nil, err
		}
		return map[string]any{"hosts": hosts, "count": len(hosts)}, nil
	}

	if env.Action == "run_windows_blender_version" {
		if env.Project != "" {
			return nil, errors.New("run_windows_blender_version does not accept a project")
		}
		var a struct {
			HostID string `json:"host_id"`
		}
		if err := decodePrivateControlArgs(env.Args, &a); err != nil {
			return nil, err
		}
		job, err := core.SubmitHostBridgeJob(strings.TrimSpace(a.HostID), core.HostJobSpec{Tool: core.HostBridgeToolBlender, Operation: core.HostBridgeOperationVersion})
		if err != nil {
			return nil, err
		}
		return map[string]any{"host_job": job}, nil
	}

	if env.Action == "get_windows_host_job" {
		if env.Project != "" {
			return nil, errors.New("get_windows_host_job does not accept a project")
		}
		var a struct {
			JobID string `json:"job_id"`
		}
		if err := decodePrivateControlArgs(env.Args, &a); err != nil {
			return nil, err
		}
		job, err := core.GetHostBridgeJob(strings.TrimSpace(a.JobID))
		if err != nil {
			return nil, err
		}
		return map[string]any{"host_job": job}, nil
	}

	if env.Action == "ensure_github_project" {
		if env.Project != "" {
			return nil, errors.New("ensure_github_project does not accept a project")
		}
		var a struct {
			Repository string `json:"repository"`
		}
		if err := decodePrivateControlArgs(env.Args, &a); err != nil {
			return nil, err
		}
		project, cloned, err := core.EnsureRunnerGitHubProject(ctx, strings.TrimSpace(a.Repository))
		if err != nil {
			return nil, err
		}
		return map[string]any{"project": project, "cloned": cloned, "runner_ready": true}, nil
	}

	if env.Action == "inspect_machine_batch" {
		if env.Project != "" {
			return nil, errors.New("inspect_machine_batch does not accept a project; it targets the configured Workbench operator host")
		}
		var a struct {
			Commands []struct {
				Program        string   `json:"program"`
				Args           []string `json:"args,omitempty"`
				TimeoutSeconds int      `json:"timeout_seconds,omitempty"`
			} `json:"commands"`
		}
		if err := decodePrivateControlArgs(env.Args, &a); err != nil {
			return nil, err
		}
		requests := make([]core.MachineCommandRequest, 0, len(a.Commands))
		for _, command := range a.Commands {
			requests = append(requests, core.MachineCommandRequest{
				Program:        strings.TrimSpace(command.Program),
				Args:           command.Args,
				TimeoutSeconds: command.TimeoutSeconds,
			})
		}
		batch, err := core.InspectMachineBatch(ctx, requests)
		if err != nil {
			return nil, err
		}
		return map[string]any{"batch": batch}, nil
	}

	if env.Action == "inspect_machine" || env.Action == "run_machine_command" {
		if env.Project != "" {
			return nil, errors.New(env.Action + " does not accept a project; it targets the configured Workbench operator host")
		}
		var a struct {
			Program        string   `json:"program"`
			Args           []string `json:"args,omitempty"`
			TimeoutSeconds int      `json:"timeout_seconds,omitempty"`
		}
		if err := decodePrivateControlArgs(env.Args, &a); err != nil {
			return nil, err
		}
		req := core.MachineCommandRequest{
			Program:        strings.TrimSpace(a.Program),
			Args:           a.Args,
			TimeoutSeconds: a.TimeoutSeconds,
		}
		var result core.MachineCommandResult
		var err error
		if env.Action == "inspect_machine" {
			result, err = core.InspectMachine(ctx, req)
		} else {
			result, err = core.RunMachineCommand(ctx, req)
		}
		if err != nil {
			message := err.Error()
			if output := strings.TrimSpace(result.Output); output != "" {
				if len(output) > 16<<10 {
					output = output[:16<<10] + "\n… error output truncated by Workbench relay …"
				}
				message += ": " + output
			}
			return nil, errors.New(message)
		}
		return map[string]any{"command": result}, nil
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
			"limit":       a.Limit,
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

	case "run_operations_script":
		var a struct {
			Path           string   `json:"path"`
			Args           []string `json:"args,omitempty"`
			Commit         string   `json:"commit,omitempty"`
			TimeoutSeconds int      `json:"timeout_seconds,omitempty"`
		}
		if err := decodePrivateControlArgs(env.Args, &a); err != nil {
			return nil, err
		}
		result, runErr := core.RunOperationsScript(ctx, project, core.OperationsScriptRequest{
			Path:           strings.TrimSpace(a.Path),
			Args:           a.Args,
			Commit:         strings.TrimSpace(a.Commit),
			TimeoutSeconds: a.TimeoutSeconds,
		})
		if runErr != nil {
			message := runErr.Error()
			if output := strings.TrimSpace(result.Output); output != "" {
				if len(output) > 16<<10 {
					output = output[:16<<10] + "\n… error output truncated by Workbench relay …"
				}
				message += ": " + output
			}
			return nil, errors.New(message)
		}
		return map[string]any{"operation_script": result}, nil

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
